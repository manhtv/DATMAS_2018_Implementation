/*
Package server is used to start a new ABCI server.

It contains two server implementation:
 * gRPC server
 * socket server

*/

package main

import (
	"fmt"
	"flag"
	"os"
	srv "github.com/racin/DATMAS_2018_Implementation/server"
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
	abciPtr := flag.String("abci", "socket", "socket | grpc")
	//storePtr := flag.String("store", "app.ldb", "store path")
	flag.Parse()

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	// Create the application - in memory or persisted to disk
	app := types.NewBaseApplication()

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
	fmt.Println("Racin har started en app! Transport: " + *abciPtr);
	// Wait forever
	common.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
}