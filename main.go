package main

/*
Run as a tcp port proxy if there are multi datacentors in your
production, Receive the traffic and redirect to real server.

cz-20151119
*/

import (
	"database/sql"
	"flag"
	"github.com/VividCortex/godaemon"
	"github.com/mlycore/log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//ignore signal
func waitSignal() {
	var sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan)
	for sig := range sigChan {
		switch sig {
		case syscall.SIGINT: fallthrough
		case syscall.SIGTERM: log.Infof("terminated by signal %v", sig); os.Exit(0)
		default:
			log.Infof("received signal: %v, ignore", sig)
		}
	}
}

//net timeout
const timeout = time.Second * 2

var Bsize uint
var Verbose bool
var Dbh *sql.DB

func init() {
	log.SetLevel(os.Getenv("LOGLEVEL"))
}

func main() {
	// options
	var bind, backend, logTo string
	var buffer uint
	var daemon bool
	var verbose bool
	var conf string

	flag.StringVar(&bind, "bind", ":8002", "locate ip and port")
	flag.StringVar(&backend, "backend", "127.0.0.1:8003", "backend server ip and port")
	flag.StringVar(&logTo, "logTo", "stdout", "stdout or syslog")
	flag.UintVar(&buffer, "buffer", 4096, "buffer size")
	flag.BoolVar(&daemon, "daemon", false, "run as daemon process")
	flag.BoolVar(&verbose, "verbose", false, "print verbose sql query")
	flag.StringVar(&conf, "conf", "", "config file to verify database and record sql query")
	flag.Parse()
	Bsize = buffer
	Verbose = verbose

	conf_fh, err := get_config(conf)
	if err != nil {
		log.Errorf("Can't get config info, skip insert log to mysql...")
	} else {
	    backend_dsn, _ := get_backend_dsn(conf_fh)
	    Dbh, err = dbh(backend_dsn)
    	if err != nil {
	    	log.Errorf("Can't get database handle, skip insert log to mysql...")
	    }
	    defer Dbh.Close()
    }

	//log.SetOutput(os.Stdout)
	//if logTo == "syslog" {
	//	w, err := syslog.New(syslog.LOG_INFO, "portproxy")
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	log.SetOutput(w)
	//}

	if daemon == true {
		godaemon.MakeDaemon(&godaemon.DaemonAttr{})
	}

	p := New(bind, backend, uint32(buffer))
	log.Infof("portproxy started.")
	go p.Start()
	waitSignal()
}
