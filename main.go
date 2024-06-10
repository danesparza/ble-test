package main

import (
	"context"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

func main() {
	//	Setup signal processing
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go handleSignals(ctx, sigs, cancel)

	//	Set log info:
	log.Logger = log.With().Timestamp().Caller().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339Nano

	namespaceUUID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("www.danesparza.net"))
	serviceUUID := uuid.NewSHA1(namespaceUUID, []byte("ble-test"))
	serviceBleUUID, _ := bluetooth.ParseUUID(serviceUUID.String())
	localNameSuffix := uuid.NewMD5(uuid.Nil, []byte(mustGetMacAddr())).String()[:8]

	//	Enable the bluetooth adapter
	err := adapter.Enable()
	if err != nil {
		log.Err(err).Msg("problem enabling adapter")
		return
	}

	err = adapter.AddService(&bluetooth.Service{
		UUID:            serviceBleUUID,
		Characteristics: []bluetooth.CharacteristicConfig{},
	})
	if err != nil {
		log.Err(err).Msg("problem adding service")
		return
	}

	//	Configure the bluetooth advertisement
	adv := adapter.DefaultAdvertisement()
	err = adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    "ApplianceMonitor - " + localNameSuffix,
		ServiceUUIDs: []bluetooth.UUID{serviceBleUUID},
	})
	if err != nil {
		log.Err(err).Msg("problem configuring advertiser")
		return
	}

	// Start advertising
	err = adv.Start()
	if err != nil {
		log.Err(err).Msg("problem starting advertiser")
		return
	}

	log.Info().Msg("ble test started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("ble test stopped")
			return
		}
	}

}

func mustGetMacAddr() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, interf := range interfaces {
		a := interf.HardwareAddr.String()
		if a != "" {
			return a
		}
	}
	panic("no MAC address found.")
}

func handleSignals(ctx context.Context, sigs <-chan os.Signal, cancel context.CancelFunc) {
	select {
	case <-ctx.Done():
	case sig := <-sigs:
		switch sig {
		case os.Interrupt:
			log.Info().Msg("SIGINT")
		case syscall.SIGTERM:
			log.Info().Msg("SIGTERM")
		}

		log.Info().Msg("Shutting down ...")
		cancel()
		os.Exit(0)
	}
}
