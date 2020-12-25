package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rifflock/lfshook"

	com_port_monitor "github.com/elemc/com-port-monitor"

	"github.com/elemc/serial"
	"github.com/sirupsen/logrus"
)

func main() {
	config := &serial.Config{RS485: serial.RS485Config{
		//Enabled: true,
		//RxDuringTx: true,
	}}

	flag.StringVar(&config.Address, "addr", "", "serial port address")
	flag.IntVar(&config.BaudRate, "baud-rate", 9600, "baud rate ")
	flag.IntVar(&config.DataBits, "data-bits", 8, "data bits (5, 6, 7, 8)")
	flag.IntVar(&config.StopBits, "stop-bit", 1, "stop bit (1 or 2)")
	flag.StringVar(&config.Parity, "parity", "N", "parity")
	flag.DurationVar(&config.Timeout, "timeout", time.Millisecond*300, "TCP timeout")
	flag.Parse()

	logger := logrus.New()
	formatter := &logrus.TextFormatter{
		FullTimestamp:    true,
		TimestampFormat:  time.RFC3339Nano,
		DisableTimestamp: false,
	}
	logger.SetFormatter(formatter)
	logger.SetLevel(logrus.DebugLevel)
	pathMap := lfshook.PathMap{
		logrus.DebugLevel: "debug.log",
		logrus.InfoLevel:  "info.log",
		logrus.WarnLevel:  "warn.log",
		logrus.ErrorLevel: "error.log",
	}
	logger.Hooks.Add(lfshook.NewHook(
		pathMap,
		formatter,
	))

	m := com_port_monitor.NewMonitor(config, logger)
	if err := m.Start(); err != nil {
		logger.Fatal(err)
	}
	go signals(m)

	<-m.StopChan()
}

func signals(m com_port_monitor.Monitor) {
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGTERM)
	signal.Notify(signalChannel, syscall.SIGINT)
	signal.Notify(signalChannel, syscall.SIGKILL)
	signal.Notify(signalChannel, syscall.SIGHUP)
	<-signalChannel

	m.Stop()
}
