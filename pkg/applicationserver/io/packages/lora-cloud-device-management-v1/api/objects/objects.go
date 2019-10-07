// Copyright © 2019 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package objects

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"go.thethings.network/lorawan-stack/pkg/errors"
	"go.thethings.network/lorawan-stack/pkg/types"
)

// Implemented as per https://www.loracloud.com/documentation/device_management?url=v1.html#object-formats

// DeviceInfo encapsulates the current state of a modem as known to the server.
type DeviceInfo struct {
	// DMPorts is an array of ports currently accepted as dmport.
	DMPorts []uint8 `json:"dmports"`
}

// InfoFields contains the value of the various information fields and the timestamp of their last update.
type InfoFields struct {
	Status *struct {
		Timestamp float64     `json:"timestamp"`
		Value     StatusField `json:"value"`
	} `json:"status"`
	Charge *struct {
		Timestamp float64 `json:"timestamp"`
		Value     uint16  `json:"value"`
	} `json:"charge"`
	Voltage *struct {
		Timestamp float64 `json:"timestamp"`
		Value     float64 `json:"value"`
	} `json:"voltage"`
	Temp *struct {
		Timestamp float64 `json:"timestamp"`
		Value     int32   `json:"value"`
	} `json:"temp"`
	Signal *struct {
		Timestamp float64 `json:"timestamp"`
		Value     struct {
			RSSI float64 `json:"rssi"`
			SNR  float64 `json:"snr"`
		} `json:"value"`
	} `json:"signal"`
	Uptime *struct {
		Timestamp float64 `json:"timestamp"`
		Value     uint16  `json:"value"`
	} `json:"uptime"`
	RxTime *struct {
		Timestamp float64 `json:"timestamp"`
		Value     uint16  `json:"value"`
	} `json:"rxtime"`
	Firmware *struct {
		Timestamp float64 `json:"timestamp"`
		Value     struct {
			FwCRC string `json:"fwcrc"`
			FwCnt uint16 `json:"fwcnt"`
		} `json:"value"`
	} `json:"firmware"`
	ADRMode *struct {
		Timestamp float64 `json:"timestamp"`
		Value     uint8   `json:"value"`
	} `json:"adrmode"`
	JoinEUI *struct {
		Timestamp float64 `json:"timestamp"`
		Value     uint8   `json:"value"`
	} `json:"joineui"`
	Interval *struct {
		Timestamp float64 `json:"timestamp"`
		Value     uint8   `json:"value"`
	} `json:"interval"`
	Region *struct {
		Timestamp float64 `json:"timestamp"`
		Value     uint8   `json:"value"`
	} `json:"region"`
	OpMode *struct {
		Timestamp float64 `json:"timestamp"`
		Value     uint32  `json:"value"`
	} `json:"opmode"`
	CrashLog *struct {
		Timestamp float64 `json:"timestamp"`
		Value     string  `json:"value"`
	} `json:"crashlog"`
	RstCount *struct {
		Timestamp float64 `json:"timestamp"`
		Value     uint16  `json:"value"`
	} `json:"rstcount"`
	DevEUI *struct {
		Timestamp float64 `json:"timestamp"`
		Value     string  `json:"value"`
	} `json:"deveui"`
	FactRst *struct {
		Timestamp float64 `json:"timestamp"`
		Value     uint16  `json:"value"`
	} `json:"factrst"`
	Session *struct {
		Timestamp float64 `json:"timestamp"`
		Value     uint16  `json:"value"`
	} `json:"session"`
	ChipEUI *struct {
		Timestamp float64 `json:"timestamp"`
		Value     uint16  `json:"value"`
	} `json:"chipeui"`
	StreamPar *struct {
		Timestamp float64   `json:"timestamp"`
		Value     StreamPar `json:"value"`
	} `json:"streampar"`
	AppStatus *struct {
		Timestamp float64 `json:"timestamp"`
		Value     string  `json:"value"`
	} `json:"appstatus"`
}

type StatusField struct {
	Brownout bool `json:"brownout"`
	Crash    bool `json:"crash"`
	Mute     bool `json:"mute"`
	Joined   bool `json:"joined"`
	Suspend  bool `json:"suspend"`
	Upload   bool `json:"upload"`
}

type StreamPar struct {
	Port    uint8 `json:"port"`
	EncMode bool  `json:"encmode"`
}

type requestParam interface {
	isRequestParam()
}

// Request encapsulates a modem request.
type Request struct {
	Type  string       `json:"type"`
	Param requestParam `json:"param,omitempty"`
}

const (
	ResetRequestType    = "RESET"
	RejoinRequestType   = "REJOIN"
	MuteRequestType     = "MUTE"
	GetInfoRequestType  = "GETINFO"
	SetConfRequestType  = "SETCONF"
	FileDoneRequestType = "FILEDONE"
	FUOTARequestType    = "FUOTA"
)

type baseRequest struct {
	Type  string          `json:"type"`
	Param json.RawMessage `json:"param"`
}

var errInvalidRequestType = errors.DefineInvalidArgument("invalid_request_type", "request type `{type}` is invalid")

// UnmarshalJSON implements json.Unmarshaler.
func (r *Request) UnmarshalJSON(b []byte) error {
	var br baseRequest
	err := json.Unmarshal(b, &br)
	if err != nil {
		return err
	}
	r.Type = br.Type
	switch br.Type {
	case ResetRequestType:
		var p ResetRequestParam
		r.Param = &p
	case RejoinRequestType, MuteRequestType:
		return nil
	case GetInfoRequestType:
		var p GetInfoRequestParam
		r.Param = &p
	case SetConfRequestType:
		var p SetConfRequestParam
		r.Param = &p
	case FileDoneRequestType:
		var p FileDoneRequestParam
		r.Param = &p
	case FUOTARequestType:
		var p FUOTARequestParam
		r.Param = &p
	default:
		return errInvalidRequestType.WithAttributes("type", br.Type)
	}
	return json.Unmarshal(br.Param, r.Param)
}

type ResetRequestParam uint8

func (ResetRequestParam) isRequestParam() {}

type GetInfoRequestParam []string

func (GetInfoRequestParam) isRequestParam() {}

type SetConfRequestParam struct {
	ADRMode  *uint8      `json:"adrmode,omitempty"`
	JoinEUI  types.EUI64 `json:"joineui,omitempty"`
	Interval *uint8      `json:"interval,omitempty"`
	Region   *uint8      `json:"region,omitempty"`
	OpMode   *uint32     `json:"opmode,omitempty"`
}

func (SetConfRequestParam) isRequestParam() {}

type FileDoneRequestParam struct {
	SID  int32 `json:"sid"`
	SCtr int32 `json:"sctr"`
}

func (FileDoneRequestParam) isRequestParam() {}

type FUOTARequestParam Hex

// MarshalJSON implements json.Marshaler.
func (f FUOTARequestParam) MarshalJSON() ([]byte, error) {
	return Hex(f).MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler.
func (f *FUOTARequestParam) UnmarshalJSON(b []byte) error {
	h := Hex{}
	err := h.UnmarshalJSON(b)
	if err != nil {
		return err
	}
	*f = FUOTARequestParam(h)
	return nil
}

func (FUOTARequestParam) isRequestParam() {}

// UploadSession contains information for an active, obsolete or finished file upload.
type UploadSession struct {
	State  string   `json:"string"`
	SID    uint8    `json:"sid"`
	SCtr   uint8    `json:"sctr"`
	CCt    uint16   `json:"cct"`
	CSz    uint8    `json:"csz"`
	Chunks []string `json:"chunks"`
}

// StreamSession contains information for a defined streaming session.
type StreamSession struct {
	Port    uint8   `json:"port"`
	Decoder *string `json:"decoder,omitempty"`
}

// LoRaUplink encapsulates the information of a LoRa message.
type LoRaUplink struct {
	FCnt      uint32  `json:"fcnt"`
	Port      uint8   `json:"port"`
	Payload   Hex     `json:"payload"`
	DR        uint8   `json:"dr"`
	Freq      uint32  `json:"freq"`
	Timestamp float64 `json:"timestamp"`
}

// LoRaDnlink is a specification for a modem device.
type LoRaDnlink struct {
	Port    uint8 `json:"port"`
	Payload Hex   `json:"payload"`
}

// File carries the contents of an uploaded, defragmented file by a modem to the service.
type File struct {
	SCtr      uint8   `json:"sctr"`
	Timestamp float64 `json:"timestamp"`
	Port      uint8   `json:"port"`
	Data      Hex     `json:"string"`
	Hash      Hex     `json:"hash"`
	EncMode   bool    `json:"encmode"`
	Message   *string `json:"message,omitempty"`
}

// Stream is a fully assembled record of the stream session stored
// in the stream record history with the device state.
type Stream struct {
	Timestamp float64 `json:"timestamp"`
	Port      uint8   `json:"port"`
	Data      Hex     `json:"data"`
	Off       uint16  `json:"off"`
}

// LogMessage is a log message associated with a device.
type LogMessage struct {
	LogMsg    string  `json:"logmsg"`
	Level     string  `json:"level"`
	Timestamp float64 `json:"timestamp"`
}

// DeviceSettings holds the initial settings for a device.
type DeviceSettings struct {
	DMPorts *int32 `json:"dmports,omitempty"`
	Streams []struct {
		Port  int32 `json:"port"`
		AppSz int32 `json:"appsz"`
	} `json:"streams,omitempty"`
}

// TokenInfo holds token information.
type TokenInfo struct {
	Name         string   `json:"name"`
	Token        string   `json:"token"`
	Capabilities []string `json:"capabilities"`
}

// Hex represents hex encoded bytes.
type Hex []byte

// MarshalJSON implements json.Marshaler.
func (h Hex) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", hex.EncodeToString(h))), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (h *Hex) UnmarshalJSON(b []byte) (err error) {
	s := strings.TrimSuffix(strings.TrimPrefix(string(b), "\""), "\"")
	*h, err = hex.DecodeString(s)
	return
}
