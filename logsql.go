package main

import (
	"fmt"
	"github.com/mlycore/log"
	"strconv"
	"strings"
)

//read more client-server protocol from http://dev.mysql.com/doc/internals/en/text-protocol.html
const (
	comQuit byte = iota + 1
	comInitDB
	comQuery
	comFieldList
	comCreateDB
	comDropDB
	comRefresh
	comShutdown
	comStatistics
	comProcessInfo
	comConnect
	comProcessKill
	comDebug
	comPing
	comTime
	comDelayedInsert
	comChangeUser
	comBinlogDump
	comTableDump
	comConnectOut
	comRegiserSlave
	comStmtPrepare
	comStmtExecute
	comStmtSendLongData
	comStmtClose
	comStmtReset
	comSetOption
	comStmtFetch
)

type query struct {
	bindPort  int64
	client    string
	cport     int64
	server    string
	sport     int64
	sqlType   string
	sqlString string
}

func ipPortFromNetAddr(s string) (ip string, port int64) {
	addrInfo := strings.SplitN(s, ":", 2)
	ip = addrInfo[0]
	port, _ = strconv.ParseInt(addrInfo[1], 10, 64)
	return
}

func converToUnixLine(sql string) string {
	sql = strings.Replace(sql, "\r\n", "\n", -1)
	sql = strings.Replace(sql, "\r", "\n", -1)
	return sql
}

func sql_escape(s string) string {
	var j int = 0
	if len(s) == 0 {
		return ""
	}

	tempStr := s[:]
	desc := make([]byte, len(tempStr)*2)
	for i := 0; i < len(tempStr); i++ {
		flag := false
		var escape byte
		switch tempStr[i] {
		case '\r':
			flag = true
			escape = '\r'
			break
		case '\n':
			flag = true
			escape = '\n'
			break
		case '\\':
			flag = true
			escape = '\\'
			break
		case '\'':
			flag = true
			escape = '\''
			break
		case '"':
			flag = true
			escape = '"'
			break
		case '\032':
			flag = true
			escape = 'Z'
			break
		default:
		}
		if flag {
			desc[j] = '\\'
			desc[j+1] = escape
			j = j + 2
		} else {
			desc[j] = tempStr[i]
			j = j + 1
		}
	}
	return string(desc[0:j])
}

func proxyLog(local, proxy *Conn) {
	buffer := make([]byte, Bsize)
	var sqlInfo query
	sqlInfo.client, sqlInfo.cport = ipPortFromNetAddr(local.conn.RemoteAddr().String())
	sqlInfo.server, sqlInfo.sport = ipPortFromNetAddr(proxy.conn.RemoteAddr().String())
	_, sqlInfo.bindPort = ipPortFromNetAddr(local.conn.LocalAddr().String())

	for {
		n, err := local.Read(buffer)
		log.Errorf("local read buffer: %d, %s", n, string(buffer))
		if err != nil {
			log.Errorf("local read buffer error: %s", err)
			return
		}
		if n >= 5 {
			var verboseStr string
			switch buffer[4] {
			case comQuit:
				verboseStr = fmt.Sprintf("From %s To %s; Quit: %s", sqlInfo.client, sqlInfo.server, "user quit")
				sqlInfo.sqlType = "Quit"
			case comInitDB:
				verboseStr = fmt.Sprintf("From %s To %s; schema: use %s", sqlInfo.client, sqlInfo.server, string(buffer[5:n]))
				sqlInfo.sqlType = "Schema"
			case comQuery:
				verboseStr = fmt.Sprintf("From %s To %s; Query: %s", sqlInfo.client, sqlInfo.server, string(buffer[5:n]))
				sqlInfo.sqlType = "Query"
			//case comFieldList:
			//	verboseStr = log.Printf("From %s To %s; Table columns list: %s", sqlInfo.client, sqlInfo.server, string(buffer[5:n]))
			//	sqlInfo.sqlType = "Table columns list"
			case comCreateDB:
				verboseStr = fmt.Sprintf("From %s To %s; CreateDB: %s", sqlInfo.client, sqlInfo.server, string(buffer[5:n]))
				sqlInfo.sqlType = "CreateDB"
			case comDropDB:
				verboseStr = fmt.Sprintf("From %s To %s; DropDB: %s", sqlInfo.client, sqlInfo.server, string(buffer[5:n]))
				sqlInfo.sqlType = "DropDB"
			case comRefresh:
				verboseStr = fmt.Sprintf("From %s To %s; Refresh: %s", sqlInfo.client, sqlInfo.server, string(buffer[5:n]))
				sqlInfo.sqlType = "Refresh"
			case comStmtPrepare:
				verboseStr = fmt.Sprintf("From %s To %s; Prepare Query: %s", sqlInfo.client, sqlInfo.server, string(buffer[5:n]))
				sqlInfo.sqlType = "Prepare Query"
			case comStmtExecute:
				verboseStr = fmt.Sprintf("From %s To %s; Prepare Args: %s", sqlInfo.client, sqlInfo.server, string(buffer[5:n]))
				sqlInfo.sqlType = "Prepare Args"
			case comProcessKill:
				verboseStr = fmt.Sprintf("From %s To %s; Kill: kill conntion %s", sqlInfo.client, sqlInfo.server, string(buffer[5:n]))
				sqlInfo.sqlType = "Kill"
			default:
			}

			if Verbose {
				log.Tracef(verboseStr)
			}

			if strings.EqualFold(sqlInfo.sqlType, "Quit") {
				sqlInfo.sqlString = "user quit"
			} else {
				sqlInfo.sqlString = converToUnixLine(sql_escape(string(buffer[5:n])))
			}

			if !strings.EqualFold(sqlInfo.sqlType, "") && Dbh != nil {
				insertlog(Dbh, &sqlInfo)
			}

		}

		log.Errorf("proxy write buffer: %s", buffer[0:n])
		_, err = proxy.Write(buffer[0:n])
		if err != nil {
			return
		}
	}
}
