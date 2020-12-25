package com_port_monitor

import (
	"encoding/hex"
	"net"
	"time"

	"github.com/elemc/serial"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Monitor interface {
	Start() error
	Stop()
	StopChan() <-chan struct{}
}

type monitor struct {
	config *serial.Config
	logger *logrus.Logger

	serialPort serial.Port

	data           chan []byte
	doneRead       chan struct{}
	doneDataListen chan struct{}
	stopChan       chan struct{}
	isStopped      bool
}

// NewMonitor - создает новый экземпляр монитора для прослушивания данных
func NewMonitor(config *serial.Config, logger *logrus.Logger) Monitor {
	return &monitor{
		config: config,
		logger: logger,
	}
}

func (m *monitor) Start() (err error) {
	if m.serialPort, err = serial.Open(m.config); err != nil {
		err = errors.Wrapf(err, "unable to open serial port: %s", m.config.Address)
		return
	}
	m.logger.Debug("Serial port open successful")

	m.data = make(chan []byte, 1000)
	go m.listenData()

	m.doneRead = make(chan struct{})
	m.doneDataListen = make(chan struct{})
	m.stopChan = make(chan struct{})

	go m.listen()
	return
}

func (m *monitor) Stop() {
	m.isStopped = true
	<-m.doneRead
	close(m.data)
	_ = m.serialPort.Close()
	m.stopChan <- struct{}{}
}

func (m *monitor) StopChan() <-chan struct{} {
	return m.stopChan
}

func (m *monitor) listen() {
	m.logger.WithField("addr", m.config.Address).Info("Start listen serial")
	buf := make([]byte, 32)
	for {
		if m.isStopped {
			break
		}
		buf = m.read(buf)
	}
	m.doneRead <- struct{}{}
}

func (m *monitor) read(gBuf []byte) []byte {
	buf := make([]byte, 16)
	count, err := m.serialPort.Read(buf)
	if err != nil && err != serial.ErrTimeout {
		if netErr, ok := err.(net.Error); ok {
			if netErr.Timeout() {
				return gBuf
			}
		}
		err = errors.Wrap(err, "unable to read from connection")
		return gBuf
	} else if err == serial.ErrTimeout {
		return gBuf
	}

	buf = buf[:count]
	gBuf = append(gBuf, buf...)

	return m.checkResponse(gBuf)
}
func (m *monitor) checkResponse(data []byte) []byte {
	if data == nil || len(data) == 0 {
		return data
	}
	for idx, char := range data {
		if char == 0x0D {
			response := data[:idx+1]
			if idx+1 == len(data) {
				data = []byte{}
			} else {
				data = data[idx+1:]
			}
			m.data <- response
			return m.checkResponse(data)
		}
	}

	return data
}

func (m *monitor) listenData() {
	for {
		if m.isStopped {
			break
		}
		select {
		case d := <-m.data:
			m.logger.
				WithField("data", string(d)).
				WithField("hex", hex.EncodeToString(d)).
				Info("Data")

		case <-time.After(m.config.Timeout * 2):
			m.logger.Warn("Timeout")
			continue
		}
	}
}
