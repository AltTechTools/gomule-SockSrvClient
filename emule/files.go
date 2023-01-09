package emule

import (
	"fmt"
	"net"
	"database/sql"
	//"errors"
	//libdeflate "github.com/4kills/go-libdeflate/v2"
)

func prconefile(filehashbuf []byte, filename string, fsize uint64, filetype string, debug bool, db *sql.DB, uhash []byte){
	var tmpbuf []byte
	if !SliceBuf(filehashbuf,0,16,&tmpbuf) {
		return
	}
	if debug {
		fmt.Println("DEBUG: user hash:", uhash) 
		fuuid := fmt.Sprintf("%x-%x-%x-%x-%x-%x-%x-%x",
		tmpbuf[0:2], tmpbuf[2:4], 
		tmpbuf[4:6], tmpbuf[6:8],
		tmpbuf[8:10], tmpbuf[10:12], 
		tmpbuf[12:14], tmpbuf[14:16])
		fmt.Println("DEBUG: File hash:", fuuid)  
		fmt.Println("DEBUG: File name:", filename)
		fmt.Println("DEBUG: File type:", filetype)
		fmt.Println("DEBUG: File size:", fsize)
	}
	//files
	res, err := db.Exec("UPDATE files SET time_offer = CURRENT_TIMESTAMP WHERE hash = ?",tmpbuf)
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
		fmt.Println("Updated file Rows: ",affectedRows)
	}
	
	if affectedRows == 0 {
		res, err = db.Exec("INSERT INTO files(hash, size, time_creation, time_offer) VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",tmpbuf,fsize)
		if err != nil {
			fmt.Println("ERROR: ",err.Error())
			return
		}
	}
	
	//sources
	res, err = db.Exec("UPDATE sources SET time_offer = CURRENT_TIMESTAMP WHERE file_hash = ? AND user_hash = ?",tmpbuf,uhash)
	if err != nil {
		fmt.Println("ERROR: ",err.Error())
		return
	}

	affectedRows, err = res.RowsAffected()
	if err != nil {
		fmt.Println("ERROR: ",err.Error())
		return
	}
	if debug {
		fmt.Println("Updated source Rows: ",affectedRows)
	}
	//todo figure out ext (file extension e.g. zip)
	if affectedRows == 0 {
		res, err = db.Exec("INSERT INTO sources(file_hash, user_hash, time_offer,name,ext,type,online) VALUES (?, ?, CURRENT_TIMESTAMP, ?, ?, ?, 1)",tmpbuf,uhash,filename,filename2ext(filename),filetype)

		if err != nil {
			fmt.Println("ERROR: ",err.Error())
			panic("fuck")
			return
		}
	}

}

func prcofferfiles(buf []byte, conn net.Conn, debug bool, blen int, db *sql.DB, uhash []byte) {
	//30 bytes more: [2 1 0 1 15 0 66 111 100 121 98 117 105 108 100 101 114 46 109 112 52 3 1 0 2 104 11 112 0 2]
	// =
	// [2 1 0 1] len[15 0 ] Bodybuilder.mp4 [3 1 0 2 104 11 112 0 2]
	
	//30 bytes more: [2 1 0 1 50 0 116 104 101 46 115 105 109 112 115 111 110 115 46 115 48 50 101 49 48 46 105 110 116 101]
	// [2 1 0 1] len 50 

	var tmpbuf []byte
	if !SliceBuf(buf,0,4,&tmpbuf) {
		return
	}
	count := ByteToInt32(tmpbuf) //cant be more than 200 by spec
	if debug {
		fmt.Println("DEBUG: prcofferfiles")
		fmt.Println("DEBUG: files:", count)
	}
	iteration := 0
	byteoffset := uint32(4)
	debugloop:=debug
	for{
		if byteoffset >= uint32(blen) {
			//if iteration > 202{
			if byteoffset != uint32(blen){
				fmt.Println("WARNING: byteoffset doesn't match buffer length", byteoffset, blen)
			}
			if int32(iteration) != count{
				fmt.Println("WARNING: offerfiles: last iteration doesn't match filecount", iteration, count)
			}
			break;
		}
		if !SliceBuf(buf,int(byteoffset),int(byteoffset)+16,&tmpbuf) {
			return
		}
		filehashbuf := tmpbuf//buf[byteoffset+0:byteoffset+16]
		//obfuscated
		//fmt.Println("DEBUG: client id:", buf[byteoffset+16:byteoffset+20])
		//fmt.Println("DEBUG: client port:", buf[byteoffset+20:byteoffset+22])
		if !SliceBuf(buf,int(byteoffset)+22,int(byteoffset)+26,&tmpbuf) {
			return
		}
		itag := ByteToInt32(tmpbuf)
		//itag := ByteToInt32(buf[byteoffset+22:byteoffset+26])
		if debugloop {
			fmt.Println("DEBUG: 1. tag count:", itag)
		}
		fname := ""
		ftype := ""
		fsize := uint64(0)

		byteoffset += 26 //after tag count
		totalreadtags, tagarr := ReadTags(int(byteoffset),buf,int(itag),debug)
		if debugloop {
			fmt.Println("DEBUG: len(tagarr)",len(tagarr))
		}
		for i := 0; i < len(tagarr); i++ {
			switch tagarr[i].NameByte {
				case 0x1:
					if tagarr[i].Type == byte(2) {
						fname = fmt.Sprintf("%s",tagarr[i].Value)
						if debugloop {
							fmt.Printf("Debug Filename Tag: %s\n",tagarr[i].Value)
						}
					}
				case 0x2:
					if tagarr[i].Type == byte(3) {
						fsize = uint64(ByteToUint32(tagarr[i].Value))
						if debugloop {
							fmt.Printf("Debug File Size Tag: %d\n",ByteToUint32(tagarr[i].Value))
						}
					}
				case 0x3:
					if tagarr[i].Type == byte(2) {
						ftype = fmt.Sprintf("%s",tagarr[i].Value)
						if debugloop {
							fmt.Printf("Debug File Type Tag: %s\n",tagarr[i].Value)
						}
					}
				case 0x3a:
					if tagarr[i].Type == byte(3) {
						fsize += uint64(ByteToUint32(tagarr[i].Value)) * uint64(0x100000000)
						if debugloop {
							fmt.Printf("Debug >32bit File Size Tag: %d\n",ByteToUint32(tagarr[i].Value))
							fmt.Printf("Debug Total File Size Tag: %d\n",fsize)							
						}
					}
				default:
					if debugloop {
						fmt.Printf("Warning: unknown tag 0x%x\n",tagarr[i].NameByte)
						fmt.Println(" ->Value: ",tagarr[i].Value)
						return //test
					}
			}
		}
		//prconefile(filehashbuf, fname, fsize, ftype, debugloop, db, uhash)
		prconefile(filehashbuf, fname, fsize, ftype, false, db, uhash)
		byteoffset+=uint32(totalreadtags)

		iteration+=1

		if debugloop {
			fmt.Println("DEBUG: new byteoffset", byteoffset)
			fmt.Println("DEBUG: next iteration", iteration)
		}
	}
	if debug {
		fmt.Printf("DEBUG: processed %d files and %d bytes\n",iteration,byteoffset)
	}
}

func offerfiles(buf []byte, protocol byte, client *SockSrvClient, debug bool, n int, db *sql.DB, uhash []byte) {
	conn:=client.Conn
	if debug {
		fmt.Println("DEBUG: Client offers Files / Keep alive")
		fmt.Printf("DEBUG: File offering protocol 0x%02x\n", protocol)
	}
	var bufcomp []byte
	if !SliceBuf(buf,1,n,&bufcomp) {
		return
	}
	//bufcomp := buf[1:n]
	if protocol == 0xd4 {
		var blen int = 0
		var err error

 		var decompressed []byte  //maybe move Decompressor creation to the creation of the connection
		//dc, err := libdeflate.NewDecompressor() //not recomended to create a new instance each, but also not possible to use the same simultaniously
		//if err != nil {
		//	fmt.Println("ERROR libdeflate:", err.Error())
		//	return
		//}
		if debug {
			fmt.Println("DEBUG: decompressing")
		}
		//blen, decompressed, err = dc.Decompress(bufcomp, nil, 1)
		//blen, decompressed, err = dc.Decompress(bufcomp, nil, 1)
		blen, decompressed, err = client.DeComp.Decompress(bufcomp, nil, 1)
		//dc.Close()
		if blen != n-1{
			fmt.Println("Warning: less bytes processed", blen)
		}
		blen=len(decompressed)
		if debug {
			fmt.Println("DEBUG: after decompressing")
		}
		if err != nil {
			fmt.Println("ERROR decompress:", err.Error())
			fmt.Println("ERROR: uncompressed len", blen)
			fmt.Println("ERROR: uncompressed buf 10", decompressed[0:10])
			return
		}
		if debug {
			fmt.Println("DEBUG: uncompressed bytes", blen)
		}
		//fmt.Println("DEBUG: uncompressed buf 10", decompressed[blen+0:blen+10])
		prcofferfiles(decompressed, conn, debug, blen, db, uhash)
	} else if protocol == 0xe3 {
		prcofferfiles(bufcomp, conn, debug, n-1, db, uhash)
	} else {
		fmt.Println("Error: offerfiles: wrong protocol")
	}
}

func filesources(buf []byte, uhash []byte, protocol byte, conn net.Conn, debug bool, n int, db *sql.DB) {
	//type=buf[0]
	var tmpbuf []byte
	if debug {
		fmt.Println("DEBUG: Client looks for File Sources")
		if !SliceBuf(buf,1,17,&tmpbuf) {
			return
		}
		fmt.Println("DEBUG: 16lehash:", tmpbuf)
		fmt.Printf("DEBUG: file hash: %x\n",tmpbuf)
		if !SliceBuf(buf,17,n,&tmpbuf) {
			return
		}
		fmt.Println("DEBUG: size bytes after hash:", tmpbuf,ByteToUint32(tmpbuf)) 
		//current db layout doesn't allow for the same hash with differing sizes (unique key)
		//thus I ignore it until I decide on a new db layout.
		//fmt.Println("DEBUG: 16revhas:", buf[n-16:n]) //not a valid hash
		//queryfilesources(buf[n-16:n],debug,db)

		//fmt.Println("DEBUG: full buf:", n, buf[0:n])
	}
	if !SliceBuf(buf,1,17,&tmpbuf) {
		return
	}
	data := make([]byte, 0)
	listitems, srcdata:=queryfilesources(tmpbuf,uhash,debug,db) //valid hash
	if listitems > 0 {
		if debug {
			fmt.Println("DEBUG: found sources: ",listitems)
			fmt.Println("DEBUG: found sources bytes: ",listitems*6)
			fmt.Println("DEBUG: found sources data: ",srcdata) //+18 (16+type+sources count) = full answersize
		}
		//protocol 0xE3, found sources type 0x42
		msgsize := uint32(listitems)*uint32(6)
		msgsize += uint32(18) //Type0x42 + file hash + sources count(1byte)
		data = append(data,protocol)
		data = append(data,UInt32ToByte(msgsize)...)
		data = append(data,0x42)
		data = append(data,tmpbuf...) //file hash
		data = append(data,byte(listitems))   // count of sources, just one byte? - limit 255 in sql querry
		data = append(data,srcdata...)
		if debug {
			fmt.Println("DEBUG: sources answer: ",data) //fmt.Println("DEBUG: sources answer: ",data[1:30])
		}
		conn.Write(data)
	} else {
		if debug {
			fmt.Println("DEBUG: found sources: None found ")
		}
	}
}

func queryfilesources(filehash []byte, uhash []byte, debug bool, db *sql.DB) (listitems int, srcdata []byte){
	srcdata = make([]byte, 0)
	listitems = 0
	srcuhash := make([]byte, 16)
	var ed2kid uint32
	var port int16 //var port uint16
	rows, err := db.Query("select sources.user_hash,clients.id_ed2k,clients.port from sources left join clients on sources.user_hash=clients.hash where sources.file_hash = ? AND sources.user_hash <> ? LIMIT 255", filehash, uhash)
	//INNER JOIN Customers ON Orders.CustomerID=Customers.CustomerID;
	if err != nil {
		fmt.Println("ERROR query: ",err.Error())
		return
	}
	for rows.Next() {
		err := rows.Scan(&srcuhash,&ed2kid,&port)
		if err != nil {
			fmt.Println("ERROR row scan: ",err.Error())
			return
		}
		listitems+=1
		bytes:=UInt32ToByte(ed2kid)
		//srcdata = append(srcdata,byte(192),byte(168),byte(1),byte(249))//
		srcdata = append(srcdata,bytes[0:4]...)
		bytes=Int16ToByte(port)
		srcdata = append(srcdata,bytes[0:2]...)
		if debug {
			fmt.Println("DEBUG: SOURCE: HASH: ",srcuhash)
			fmt.Println("DEBUG: SOURCE: ed2kid: ",ed2kid)
			fmt.Println("DEBUG: SOURCE: port: ",port)
		}
	}
	err = rows.Err()
	if err != nil {
		fmt.Println("ERROR rows: ",err.Error())
	}
	rows.Close()
	if debug && listitems > 0 {
		var fsize uint64 //info only
		err = db.QueryRow("select size from files where hash = ?", filehash).Scan(&fsize)
		if err != nil {
			fmt.Println("DEBUG: filehash ERROR: ",err.Error())
		}
		fmt.Println("DEBUG: SOURCE: file size: ",fsize)
		//fmt.Println("DEBUG: SOURCE: file size: ",UInt32ToByte(fsize))
	}
	return
}

func listservers(buf []byte, protocol byte, conn net.Conn, debug bool, n int) {
	//type=buf[0]
	if debug {
		fmt.Println("DEBUG: Get list of servers")
		//fmt.Println("DEBUG: listservers")
	}
}
/*
func dbsearchfiles(query string,strarr []string, db *sql.DB){
    //params := make([]any,len(strarr)) ///test
  //for i:=0;i < len(strarr);i++ {
//	  params=append(params,strarr[i])
  //}
  //rows, err := db.Query(query,params...)
  params := make([]interface{}, 0)
  for i:=0;i < len(strarr);i++ {
	  params=append(params,strarr[i])
  }
  dbsearchfilesexec(query,&params,db)
}
*/


func dbsearchfilesexec(query string, params *[]interface{}, db *sql.DB, conn net.Conn){
	var scount int
	var sname string
	var sext string
	var stype string
	var srating int
	var sfilehash []byte
	var sfilesize uint
	fmt.Println("Debug: dbsearchfilesexec: ",query)
	fmt.Println("Debug: params: ",*params)
	rows, err := db.Query(query,*params...)
	if err != nil {
		fmt.Println("ERROR: ",err.Error())
		return
	}
	response_b := make([]byte,0)
	rescount := 0
	for rows.Next() {
		err := rows.Scan(&scount,&sname,&sext,&stype,&srating,&sfilehash,&sfilesize)
		if err != nil {
			fmt.Println("ERROR: ",err.Error())
			return
		}
		rescount++
		fmt.Println("Debug: file found: ",sname,scount)
		fmt.Printf("Debug file hash: %x, size: %d\n",sfilehash,sfilesize)
		/*
		answer:
		*/
		response_b = append(response_b,sfilehash...) //hash
		response_b = append(response_b,255,255,255,255) //clientid
		response_b = append(response_b,255,255) //clientport
		response_b = append(response_b,UInt32ToByte(uint32(4))...) //tag count
		filename_b := EncodeByteTagString(EncodeByteTagNameInt(0x1),sname)
		filesize_b := EncodeByteTagInt(EncodeByteTagNameInt(0x2),uint32(sfilesize)) //if over uint32 max add the special tag (prob like in offer files)
		filetype_b := EncodeByteTagString(EncodeByteTagNameInt(0x3),stype)
		filesources_b := EncodeByteTagInt(EncodeByteTagNameInt(0x15),uint32(scount))
		response_b = append(response_b,filename_b...)
		response_b = append(response_b,filesize_b...)
		response_b = append(response_b,filetype_b...)
		response_b = append(response_b,filesources_b...)
	}
	response2_b := make([]byte,0)
	response2_b = append(response2_b,0xe3)
	response2_b = append(response2_b,UInt32ToByte(uint32(len(response_b)+5))...) //res count 4 + 1 b type //shouldnt be too long
	response2_b = append(response2_b,0x33) // 0x16) 16 is wrong
	response2_b = append(response2_b,UInt32ToByte(uint32(rescount))...)
	response2_b = append(response2_b,response_b...)
	fmt.Println("DEBUG: search response: ",response2_b)
	conn.Write(response2_b) //respond
	return
}

func searchfiles(buf []byte, protocol byte, conn net.Conn, debug bool, n int, db *sql.DB) {
	//select name, ext, type, rating from sources WHERE name like "%a%" and name like "%three%" and name like "%10%" LIMIT 100
	/*//type=buf[0]
	//[1 4 0 116 101 115 116] //simple
	
	//starts with and (0x0 0x0) 
	//0x0 0x0 = AND
	//0x100 = OR
	//0x200 = NOT
	and (1 4 0 116 101 115 116)(2 5 0 73 109 97 103 101)
	[0 0 1 4 0 116 101 115 116 2 5 0 73 109 97 103 101 1 0 3
	
	[0 0 1 4 0 116 101 115 116 2 5 0 73 109 97 103 101 1 0 3] //typ image
	
	AND ( 1 4 0 116 101 115 116)( AND (2 5 0 73 109 97 103 101 1 0 3) (2 3 0 106 112 103 1 0 4)) -> 
	( 1 4 0 116 101 115 116) AND ( (2 5 0 73 109 97 103 101 1 0 3) AND (2 3 0 106 112 103 1 0 4) 7)
	
	[0 0 1 4 0 116 101 115 116 0 0 2 5 0 73 109 97 103 101 1 0 3 2 3 0 106 112 103 1 0 4] //image + endung jpg
	[0 0 1 4 0 116 101 115 116 2 3 0 106 112 103 1 0 4] // endung jpg
	//max search	
	
	[0 0 1 ]
	[4 0] [116 101 115 116] 
	[0 0 2]
	[5 0] [73 109 97 103 101] [1 0 3] 
	[0 0 3] 0 0 16 0 1 1 0 2 0 0 3 0 0 160 0 2 1 
	0 2 0 0 3 1 0 0 0 1 1 0 21 2 
	[3 0] [106 112 103] [1 0 4]
  
	[0 0 1] 
	[4 0] [116 101 115 116] 
	[0 0 2] 
	[5 0] [73 109 97 103 101] [1 0 3]
	[0 0 3] 0 0 16 0 1 1 0 2 0 0 3 0 0 160 0 2 1 
	[0 2 2]
	[3 0] [106 112 103] [1 0 4]
	
	*/
	//max emule 
	//[0 0 1 6 0 116 101 115 116 32 50 0 0 2 4 0 120 50 54 53 1 0 213 0 0 3 20 0 0 0 3 1 0 212 0 0 3 1 0 0 0 3 1 0 48 
	//0 0 2 3 0 80 114 111 1 0 3 0 0 3 0 0 16 0 3 1 0 2 0 0 3 0 0 144 0 4 1 0 2 0 0 3 1 0 0 0 3 1 0 21 2 3 0 106 112 103 1 0 4]
	//("test 2" type: cdimage, min size 1, max size 9, avialbility 1, complete sources 2, ext jpg, )
	
	//[0 0 1 6 0 116 101 115 116 32 50 0 0 2 4 0 120 50 54 53 1 0 213 0 0 3 90 0 0 0 3 1 0 211 
	//0 0 3 20 0 0 0 3 1 0 212 0 0 3 1 0 0 0 3 1 0 48 0 0 3 0 0 16 0 3 1 0 2 
	//0 0 3 0 0 144 0 4 1 0 2 0 0 3 1 0 0 0 3 1 0 21 2 3 0 106 112 103 1 0 4]
	//("test 2" type: any, min size 1, max size 9, avialbility 1, complete sources 2, ext jpg, codec x265, min bitrate 20, min len 00:01:30)
	/*constraint types
	1 0 213 = codec
	3 1 0 211 = max size
	3 1 0 212 = bitrate?
	3 1 0 48 = min size? /avail
	3 1 0 2 = ?
	4 1 0 2 = duration ?
	3 1 0 21 = avail ? / min size?
	1 0 4 = file ending ?
	string constriant -> 3 byte designator
	number constriant -> 4 byte desginator
	*/
	//if debug {
	//if 1==1 {
	fmt.Println("DEBUG: Client looks for Files")
	//fmt.Println("DEBUG: searchfiles")

	var tmpbuf []byte
	if !SliceBuf(buf,1,n,&tmpbuf) {
		fmt.Println("DEBUG: searchfiles: slice to n failed:", 1, n)
		return
	}
	fmt.Println("DEBUG: buf full query:", tmpbuf)
	if(tmpbuf[0] == 0x01) {
		fmt.Println("DEBUG: simple search")    
		strlen := ByteToInt16(tmpbuf[1:3])
		fmt.Println("DEBUG: strlen:", strlen)
		fmt.Println("DEBUG: strlen buf:", tmpbuf[1:3])
		if !SliceBuf(buf,4,4+int(strlen),&tmpbuf) {
			fmt.Println("DEBUG: searchfiles:slice failed:", 4, 4+strlen)
			return
		}
		fmt.Println("DEBUG: buf string:", tmpbuf)
		strbuf := tmpbuf
		str := fmt.Sprintf("%s",strbuf)
		fmt.Println("DEBUG: str:", str)
		if !SliceBuf(buf,4+int(strlen),n,&tmpbuf) {
			fmt.Println("DEBUG: searchfiles:slice failed:", 4+strlen,n)
			return
		}
		fmt.Println("DEBUG: buf other:", tmpbuf)
		//func search2query2(search string,params *[]interface{})(sqlquery string){
		//    querystr, strarr := search2query(str)
		params := make([]interface{}, 0)
		querystr :=  search2query2(str,&params)
		fmt.Println("DEBUG: qry:", querystr)
		//fmt.Println("DEBUG: strarr:", strarr)
		//dbsearchfiles(querystr,strarr,db)
		dbsearchfilesexec(querystr,&params,db,conn)
	} else {
		fmt.Println("DEBUG: complex search")
		//readConstraints(pos int, buf []byte)(readb int,ret *Constraint)
		readbytes, constraints := readConstraints(0, tmpbuf) //readConstraints(1, buf)
		fmt.Println("read bytes:",readbytes)
		if constraints == nil {
			fmt.Println("ERROR: No Contrainsts could be parsed")
			return
		}
		fmt.Println("constrain: ",constraints)
		fmt.Println("constraint type(should be and):",constraints.Type)
		if constraints.Type == C_NONE {
			fmt.Println("Type IS C_NONE")
		} else {
			fmt.Println("Type IS NOT C_NONE")
		}
		params := make([]interface{}, 0)
		//fmt.Println(stringifyConstraint(constraints, &params))
		sqlquery := constraintsearch2query(constraints, &params)
		fmt.Println(sqlquery)
		fmt.Println("params: ",params)
		dbsearchfilesexec(sqlquery,&params,db,conn)
	    
		/*
		fmt.Println("sub constraint left type(should be Main):",constraints.Left.Type)
		fmt.Println("sub constraint left type(could be something likeFileext):",constraints.Right.Type)
		fmt.Println("constraint root value",constraints.Value)
		fmt.Println("constraint Main value",constraints.Left.Value)
		fmt.Println("constraint 2nd AND value",constraints.Right.Value)

		fmt.Println("constraint 2nd AND left value",constraints.Right.Left.Value)
		fmt.Println("constraint 2nd AND right value",constraints.Right.Right.Value)

		fmt.Println("constraint 2nd AND left type",constraints.Right.Left.Type)
		fmt.Println("constraint 2nd AND right type",constraints.Right.Right.Type)
		*/
		/*
		strlen := ByteToInt16(buf[4:6])
				fmt.Println("DEBUG: strlen:", strlen)
				fmt.Println("DEBUG: strlen buf:", buf[4:6])
		fmt.Println("DEBUG: buf string:", buf[6:6+strlen])
		strbuf := buf[6:6+strlen]
				str := fmt.Sprintf("%s",strbuf)
		fmt.Println("DEBUG: str:", str)
				fmt.Println("DEBUG: buf other:", buf[6+strlen:n])
		querystr, strarr := search2query(str)
		fmt.Println("DEBUG: qry:", querystr)
		fmt.Println("DEBUG: strarr:", strarr)
		dbsearchfiles(querystr,strarr,db)
		*/
		//}
		//fmt.Println("DEBUG: buf query:", strarr)
		//buf query: [1 5 0 101 109 117 108 101]
		//emule, len 5
		//DEBUG: buf full query: [1 5 0 101 109 117 108 101]
		//DEBUG: strlen: 5
		//DEBUG: strlen buf: [5 0]
		//DEBUG: buf full query: [0 0 1 5 0 101 109 117 108 101 2 3 0 68 111 99 1 0 3]
		//emule + type texts
		//search emule with type texts
		//[0 0 1 5 0 101 109 117 108 101 2 3 0 68 111 99 1 0 3]

		//buf other: [2 3 0 68 111 99 1 0 3]
		//68 111 99 -> what is that? ASCII: "Doc"
		//search with type text and ending pdf: other:
		//[0 0 2 3 0 68 111 99 1 0 3 2 3 0 112 100 102 1 0 4]
		//[0 0 2] -> ?
		//[3 0 68 111 99] -> Doc, len 3
		//[1 0 3] -> ? , also in search with type only
		//[2] -> ?
		//[3 0 112 100 102] -> pdf, len 3
		//[1 0 4] -> ?
	}
}
/*
func stringifyConstraint(in *Constraint, params *[]interface{})(ret string){
	switch in.Type {
		case C_AND:
			ret = "("+stringifyConstraint(in.Left,params)+") AND ("+stringifyConstraint(in.Right,params)+")"
		case C_OR:
			ret = "("+stringifyConstraint(in.Left,params)+") OR ("+stringifyConstraint(in.Right,params)+")"
		case C_NOT:
			ret = "("+stringifyConstraint(in.Left,params)+") NOT ("+stringifyConstraint(in.Right,params)+")"
		case C_MAIN:
			//ret = fmt.Sprintf("sources.name like '%s'",in.Value)
			strarr := strings.Split(fmt.Sprintf("%s",in.Value)," ")
			ret = "("
  			for i := 0; i < len(strarr); i++ {
				if i != 0 {
					ret += " AND "
				}
				ret += "sources.name like ? "
				*params = append(*params,strarr[i])
			}
			ret += ")"
		case C_CODEC:
		case C_MINSIZE:
		case C_MAXSIZE:
		case C_FILETYPE:
			*params = append(*params,fmt.Sprintf("%s",in.Value))
			//ret = fmt.Sprintf("sources.type = '%s'",in.Value)
			ret = "sources.type = ?"
		case C_FILEEXT:
			*params = append(*params,fmt.Sprintf("%s",in.Value))
			//ret = fmt.Sprintf("sources.ext = '%s'",in.Value)
			ret = "sources.ext like ?"
		default:
			fmt.Println("ERROR: undefined Constraint Type", in.Type)
	}
	return
}
*/

func requestcallback(buf []byte, protocol byte, conn net.Conn, debug bool, n int) {
	//type=buf[0]
	if debug {
		fmt.Println("DEBUG: Client looks for another to callback")
	}
}

func udpfilesources(buf []byte, protocol byte, conn net.Conn, debug bool, n int) {
	//type=buf[0]
	if debug {
		fmt.Println("DEBUG: UDP Client looks for File Sources")
	}
}
