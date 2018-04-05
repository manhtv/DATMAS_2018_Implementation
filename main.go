package main

import (
	"fmt"
	"flag"
	"os"

	srv "github.com/racin/DATMAS_2018_Implementation/server"
	"github.com/racin/DATMAS_2018_Implementation/app"

	"github.com/tendermint/abci/types"
	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/abci/server"
	"github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"



)

func NewServer(protoAddr, transport string, app types.Application) (cmn.Service, error) {
	var s cmn.Service
	var err error
	switch transport {
	case "socket":
		s = srv.NewSocketServer(protoAddr, app)
	case "grpc":
		s = srv.NewGRPCServer(protoAddr, types.NewGRPCApplication(app))
	default:
		err = fmt.Errorf("Unknown server type %s", transport)
	}
	return s, err
}

func main(){
	addrPtr := flag.String("addr", "tcp://0.0.0.0:46658", "Listen address")
	abciPtr := flag.String("abci", "grpc", "grpc | socket")
	uploadAddrPtr := flag.String("uploadaddr", ":46659", "Upload address")
	//storePtr := flag.String("store", "app.ldb", "store path")
	flag.Parse()

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	// Create the application - in memory or persisted to disk
	app := app.NewApplication(*uploadAddrPtr)

	// Start the listener
	srv, err := server.NewServer(*addrPtr, *abciPtr, app)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	srv.SetLogger(logger.With("module", "abci-server"))
	if err := srv.Start(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	// Start the handler for uploading files.
	app.StartUploadHandler()

	fmt.Println("Racin har started en app! Transport: " + *abciPtr);
	fmt.Println("Info om app: " + app.Info(types.RequestInfo{Version: "123"}).Data)

	// Wait forever
	common.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
}
