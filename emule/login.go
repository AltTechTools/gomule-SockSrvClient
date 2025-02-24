/*
 * Copyright (C) 2013 Deepin, Inc.
 *               2013 Leslie Zhai <zhaixiang@linuxdeepin.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package emule

import (
	"fmt"
	"net"
	"database/sql"
)

func logout(uhash []byte, debug bool, db *sql.DB){
	res, err := db.Exec("UPDATE clients SET online = 0 WHERE hash = ?",uhash)
	if err != nil {
		fmt.Println("ERROR: ",err.Error())
		return
	}
	if debug {
		affectedRows, err := res.RowsAffected()
		if err != nil {
			fmt.Println("ERROR: ",err.Error())
			return
		}
		fmt.Println("Updated Rows: ",affectedRows)
	}
}

func login(buf []byte, protocol byte, conn net.Conn, debug bool, db *sql.DB, shighid uint32, sport uint16, ssname string, ssdesc string, ssmsg string, sflags uint32) (uhash []byte){ //func login(buf []byte, protocol byte, conn net.Conn, debug bool, db *sql.DB) (high_id uint32, port int16, uhash []byte){
	if debug {
		fmt.Println("DEBUG: Login")
	}
	if !SliceBuf(buf,1,17,&uhash) {
		conn.Close()
		return
	}
	//uhash=buf[1:17]
	//uhash = make([]byte, 16)
	//i := 0
	//for{
	//uhash[i] = buf[i+1]
	//i+=1
	//	if i >= 16{
	//		break
	//	}
	//}
	//buf[1:17]

	var tmpbuf []byte
	
	high_id := HighId(conn.RemoteAddr().String())
	if !SliceBuf(buf,21,23,&tmpbuf) {
		conn.Close()
		return
	}
	port := ByteToInt16(tmpbuf)
	if !SliceBuf(buf,23,27,&tmpbuf) {
		conn.Close()
		return
	}
	tags := ByteToInt32(tmpbuf)
	if debug {
		uuid := fmt.Sprintf("%x-%x-%x-%x-%x-%x-%x-%x",
		buf[1:3], buf[3:5], buf[5:7], buf[7:9], buf[9:11], buf[11:13],
		buf[13:15], buf[15:17])
		fmt.Println("DEBUG: highid:", high_id)
		fmt.Println("DEBUG: uuid:  ", uuid)
		fmt.Println("DEBUG: port:  ", port)
		fmt.Println("DEBUG: tagscount:  ", tags)
		fmt.Println("DEBUG: port bytes:  ", buf[21:23])
		fmt.Println("DEBUG: tagscount bytes:  ", buf[23:27])
		if !SliceBuf(buf,27,31,&tmpbuf) {
			conn.Close()
			return
		}
		fmt.Println("DEBUG: pre str tag bytes:  ", buf[27:31])
		//fmt.Println("DEBUG: other:  ", buf[27:50])
		//+4 some codes    [2 1 0 1 21 0 104 116
		//21 0 = lenght, 104 116 .. string
		if !SliceBuf(buf,31,33,&tmpbuf) {
			conn.Close()
			return
		}
		strlen := ByteToInt16(tmpbuf)
		if !SliceBuf(buf,33,33+int(strlen),&tmpbuf) {
			conn.Close()
			return
		}
		str := fmt.Sprintf("%s",tmpbuf)
		fmt.Println("DEBUG: user name:  ", str)
		if !SliceBuf(buf,33+int(strlen),33+int(strlen)+8,&tmpbuf) {
			conn.Close()
			return
		}
		fmt.Println("DEBUG: vers tag:  ", tmpbuf)
		if !SliceBuf(buf,33+int(strlen)+8,33+int(strlen)+16,&tmpbuf) {
			conn.Close()
			return
		}
		fmt.Println("DEBUG: port tag:  ", tmpbuf)
		if !SliceBuf(buf,33+int(strlen)+16,33+int(strlen)+24,&tmpbuf) {
			conn.Close()
			return
		}
		fmt.Println("DEBUG: flag tag:  ", tmpbuf)
		//strlen + 3*8bytes should exactly be the end of the buffer //confirmed
	}
	
	//(pos int, buf []byte, tags int)(totalread int, ret []*OneTag)
	
	totalread, tagarr := ReadTags(27,buf,int(tags),debug)
	if debug {
		fmt.Println("DEBUG: len(tagarr)",len(tagarr))
	}
	for i := 0; i < len(tagarr); i++ {
		switch tagarr[i].NameByte {
			case 0x1:
				if tagarr[i].Type == byte(2) {
					if debug {
						fmt.Printf("Debug Name Tag: %s\n",tagarr[i].Value)
					}
				}
			case 0x11:
				if debug {
					fmt.Printf("Debug Version Tag: %d\n",ByteToUint32(tagarr[i].Value))
				}
			case 0x20:
				if debug {
					fmt.Printf("Debug Flags Tag: %b\n",ByteToUint32(tagarr[i].Value))
				}
			case 0x0f:
				if debug {
					fmt.Printf("Debug Port Tag: %d\n",ByteToUint32(tagarr[i].Value))
				}
			case 0x60:
				if debug {
					fmt.Printf("Debug ipv6 Tag: %d\n",tagarr[i].Value)
				}
			default:
				if debug {
					fmt.Printf("Warning: unknown tag 0x%x\n",tagarr[i].NameByte)
					fmt.Println(" ->Value: ",tagarr[i].Value)
				}
		}
		/*fmt.Println("DEBUG: test val len:  ",tagarr[i].ValueLen)
		if tagarr[i].Type == byte(2) {
			fmt.Printf("Debug %s",tagarr[i].Value)
		}
		*/
	}
	if debug {
		fmt.Println("DEBUG: totalread:  ",totalread)
		fmt.Println("DEBUG: after loop")
	}
	/*index:=27
	tstbread, tstres := readTag(index,buf)
	index+=tstbread
	fmt.Println("DEBUG: test read name:  ",tstres.Value,tstbread)

	tstbread, tstres = readTag(index,buf)
	index+=tstbread
	fmt.Println("DEBUG: test read vers:  ",tstres.Value,tstbread)
	
	tstbread, tstres = readTag(index,buf)
	index+=tstbread
	fmt.Println("DEBUG: test read port:  ",tstres.Value,tstbread)
	
	tstbread, tstres = readTag(index,buf)
	index+=tstbread
	fmt.Println("DEBUG: test read flag:  ",tstres.Value,tstbread)
	*/
	
	res, err := db.Exec("UPDATE clients SET id_ed2k = ?, ipv4 = ?, port = ?, online = 1, time_login = CURRENT_TIMESTAMP WHERE hash = ?",high_id,high_id,port,uhash)
	if err != nil {
		fmt.Println("ERROR: ",err.Error())
		return
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		fmt.Println("ERROR: ",err.Error())
		return
	}
	if debug {
		fmt.Println("Updated Rows: ",affectedRows)
	}
	
	if affectedRows == 0 {
		res, err = db.Exec("INSERT INTO clients(hash, id_ed2k, ipv4, port, online) VALUES (?, ?, ?, ?, ?)",uhash,high_id,high_id,port,1)
	}
	if err != nil {
		fmt.Println("ERROR: ",err.Error())
		return
	}

	data := EncodeByteMsg(protocol,0x38,EncodeByteString(ssmsg))
	//"server version 0.0.1 (gomule)\nwarning - warning you\nHeLlo Brother in christ\n->New Line"))
	if debug {
		fmt.Println("DEBUG: login:", data)
	}
	conn.Write(data)

	//tcp tags uin32 here
	high_id_b := UInt32ToByte(high_id)
	tcpflags_b:= UInt32ToByte(sflags)
	data = EncodeByteMsg(protocol,0x40,[]byte{high_id_b[0],high_id_b[1],high_id_b[2],high_id_b[3],tcpflags_b[0], tcpflags_b[1], tcpflags_b[2], tcpflags_b[3]})
	if debug {
		fmt.Println("DEBUG: login:", data)
	}
	conn.Write(data)
	
	fcount_b := UInt32ToByte(readRowUint32("select count(*) from files",db))
	ucount_b := UInt32ToByte(readRowUint32("select count(*) from clients",db))
	data = EncodeByteMsg(protocol,0x34,[]byte{ucount_b[0], ucount_b[1], ucount_b[2], ucount_b[3], fcount_b[0], fcount_b[1], fcount_b[2], fcount_b[3]})
	if debug {
		fmt.Println("DEBUG: login:", data)
	}
	conn.Write(data)
	//0x41 server identification missing
	serverip_b:=UInt32ToByte(shighid)
	serverport_b:=UInt16ToByte(sport)
	serverguid_b := make([]byte,16)
	tagcount_b := UInt32ToByte(uint32(2)) //maybe not acctually honored
	iddata := make([]byte,0)
	
	iddata=append(iddata,serverguid_b...)
	iddata=append(iddata,serverip_b...)
	iddata=append(iddata,serverport_b...)
	iddata=append(iddata,tagcount_b...)
	servname := EncodeByteTagString(EncodeByteTagNameInt(0x1),ssname)
	//"Servername")
	servdesc := EncodeByteTagString(EncodeByteTagNameInt(0xb),ssdesc)
	//"Serverdesc")
	iddata=append(iddata,servname...)
	iddata=append(iddata,servdesc...)
	if debug {
		fmt.Println("DEBUG: serverguid_b:", serverguid_b)
		fmt.Println("DEBUG: serverip_b:", serverip_b)
		fmt.Println("DEBUG: serverport_b:", serverport_b)
		fmt.Println("DEBUG: tagcount_b:", tagcount_b)
		fmt.Println("DEBUG: servdesc:", servdesc)
	}
	
	data = EncodeByteMsg(protocol,0x41,iddata)
	if debug {
		fmt.Println("DEBUG: data:", data)
	}
	conn.Write(data)
	return
}
