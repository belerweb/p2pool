package main

import (
	"net"
	"net/http"
	"os"
	"os/signal"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/gorilla/mux"
	"github.com/siapool/p2pool/api"
	"github.com/siapool/p2pool/sharechain"
	"github.com/siapool/p2pool/siad"
)

func main() {

	app := cli.NewApp()
	app.Name = "Siapool node"
	app.Version = "0.1-Dev"

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	var debugLogging bool
	var bindAddress, apiAddr, rpcAddr string
	var poolFee int

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "debug, d",
			Usage:       "Enable debug logging",
			Destination: &debugLogging,
		},
		cli.StringFlag{
			Name:        "bind, b",
			Usage:       "Pool bind address",
			Value:       ":9985",
			Destination: &bindAddress,
		},
		cli.IntFlag{
			Name:        "fee, f",
			Usage:       "Pool fee, in 0.01%",
			Value:       200,
			Destination: &poolFee,
		},
		cli.StringFlag{
			Name:  "api-addr",
			Value: "localhost:9980", Usage: "which host:port the API server listens on",
			Destination: &apiAddr,
		},
		cli.StringFlag{
			Name:        "rpc-addr",
			Value:       ":9981",
			Usage:       "which port the gateway listens on",
			Destination: &rpcAddr,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.Infoln(app.Name, "-", app.Version)
		if debugLogging {
			log.SetLevel(log.DebugLevel)
			log.Debugln("Debug logging enabled")
		}
		return nil
	}

	app.Action = func(c *cli.Context) {
		// Print a startup message.
		log.Infoln("Loading...")

		// Create the listener for the server
		l, err := net.Listen("tcp", bindAddress)
		if err != nil {
			log.Fatal("Error listening on", bindAddress, err)
		}

		dc := &siad.Siad{RPCAddr: rpcAddr, APIAddr: apiAddr}
		err = dc.Start()
		if err != nil {
			log.Fatal("Error running embedded siad: ", err)
		}

		log.Infoln("Loading sharechain...")
		sc, err := sharechain.New(dc, "p2pooldata/sharechain")
		if err != nil {
			log.Fatal("Error initializing sharechain: ", err)
		}
		poolapi := api.PoolAPI{Fee: poolFee, ShareChain: sc}
		r := mux.NewRouter()
		r.Path("/fee").Methods("GET").Handler(http.HandlerFunc(poolapi.FeeHandler))
		r.Path("/version").Methods("GET").Handler(http.HandlerFunc(poolapi.VersionHandler))

		// stop the server if a kill signal is caught
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, os.Kill)
		go func() {
			<-sigChan
			log.Infoln("\rCaught stop signal, quitting...")
			dc.Close()
			l.Close()
		}()
		log.Infoln("Listening for miner requests")
		srv := &http.Server{
			Handler: r,
		}
		srv.Serve(l)

	}

	app.Run(os.Args)
}
