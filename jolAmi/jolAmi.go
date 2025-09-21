package jolAmi

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/ivahaev/amigo"
	"jahaninia.ir/cloud/asterisk/pbx/jolRestApi"
)

type ChannelInfo struct {
	callerId int
	Caller   string
	Callee   string
	LinkedID string
}
type jolAmi struct {
	wg              *sync.WaitGroup
	ami             *amigo.Amigo
	callStore       sync.Map
	debug           bool
	token           string
	urlCall         string
	urlAnswer       string
	urlReject       string
	urlEndCall      string
	lengthExtension int
	peers           *sync.Map
	pool            int
}

func NewJolAmi(host, port, username, password, token, urlAnswer, urlReject, urlEndCall, urlCall string, lengthExtension, pool int, debug bool) *jolAmi {
	settings := &amigo.Settings{Host: host, Port: port, Username: username, Password: password}
	return &jolAmi{
		ami:             amigo.New(settings),
		debug:           debug,
		peers:           &sync.Map{},
		urlReject:       urlReject,
		urlEndCall:      urlEndCall,
		urlAnswer:       urlAnswer,
		urlCall:         urlCall,
		token:           token,
		lengthExtension: lengthExtension,
		pool:            pool,
	}

}

// func (c *CallStore) Add(callID, callerID string) {
// 	c.calls.Store(callID, callerID)
// }

// func (c *CallStore) Get(callID string) (string, bool) {
// 	if v, ok := c.calls.Load(callID); ok {
// 		return v.(string), true
// 	}
// 	return "", false
// }

//	func (c *CallStore) Remove(callID string) {
//		c.calls.Delete(callID)
//	}
func eventsProcessor(e map[string]string, j *jolAmi) {

	switch e["Event"] {
	case "DialBegin":
		j.DialBeginHandle(e)
	case "Newstate":
		j.NewstateHandle(e)
	case "BridgeEnter":
		j.BridgeEnterHandle(e)
	// case "BridgeLeave":
	// 	BridgeLeaveHandle(e, config.UrlReject, config.Token, config.Length, config.Debug)
	case "Hangup":
		j.HangupHandle(e)
	case "Cdr":
		j.EndCallHandle(e)
	case "Newchannel":
		j.NewchannelHandle(e)
	default:
		fmt.Printf("Default: {{ %s }}\n", e)
	}
}

func (j *jolAmi) Run(wg *sync.WaitGroup) {
	defer func() {
		fmt.Println("done")
		wg.Done()
	}()
	//wg.Wait()
	j.ami.On("connect", func(message string) {
		fmt.Println("Connected", message)
	})
	j.ami.On("error", func(message string) {
		fmt.Println("Connection error:", message)
	})

	j.ami.Connect()

	c := make(chan map[string]string, j.pool)
	j.ami.SetEventChannel(c)

	for {
		go eventsProcessor(<-c, j)
	}

}

func (j *jolAmi) DialBeginHandle(data map[string]string) {
	j.DebugCheck("DialBeginHandle: {{ %s }}\n", data)
	// ***
	uniqueid, ok := GetUniqueID(data)
	if !ok {
		return
	}
	if loaded, ok := j.callStore.Load(uniqueid); ok && data["DestExten"] != "s" {
		call := loaded.(*ChannelInfo)
		j.DebugCheck("Link: %s => DestCallerIDNum:%s, DestExten:%s, DialString:%s \n", call.LinkedID, data["DestCallerIDNum"], data["DestExten"], data["DialString"])
		if data["Context"] == "ext-queues" {
			call.Callee = data["DestExten"]
		} else {
			call.Callee = data["DialString"]
		}
		j.callStore.Store(data["Uniqueid"], call)

		var jsonData = j.OutputByte("{\"ID\": \"%s\",\"STS\": \"1\" ,\"SRC\": \"%s\",\"DAKHELI\": \"%s\"}", data["Linkedid"], call.Caller, call.Callee)
		var headers = map[string]string{"accept": "*/*", "Content-Type": "application/json", "Access-token": j.token}
		jolRestApi.SendRequestJSON(j.urlCall, jsonData, headers)
	}

	// ***

	// callerID := len(data["CallerIDNum"])
	// var called, calling string

	// if ((len(data["DestCallerIDNum"]) == length || len(data["DestExten"]) == length) && callerID > length || callerID == length && len(data["DestExten"]) > length) && (data["Linkedid"] == data["Uniqueid"] || data["Linkedid"] == data["DestLinkedid"]) && (data["ChannelState"] == "6") {
	// 	if callerID > length {
	// 		called = Normalization(data["CallerIDNum"])
	// 		calling = data["DestExten"]
	// 	} else {
	// 		called = Normalization(data["DestExten"])
	// 		calling = data["CallerIDNum"]

	// 	}
	// 	var jsonData = OutputByte(debug, "{\"ID\": \"%s\",\"STS\": \"0\" ,\"SRC\": \"%s\",\"DAKHELI\": \"%s\"}", data["Linkedid"], called, calling)
	// 	var headers = map[string]string{"accept": "*/*", "Content-Type": "application/json", "Access-token": token}
	// 	jolRestApi.SendRequestJSON(urlAddr, jsonData, headers)
	// }

}

func (j *jolAmi) NewstateHandle(data map[string]string) {
	j.DebugCheck("NewstateHandle: {{ %s }}\n", data)
	callerID := len(data["CallerIDNum"])
	var called, calling string
	if (callerID == j.lengthExtension && len(data["ConnectedLineNum"]) > j.lengthExtension || len(data["ConnectedLineNum"]) == j.lengthExtension && callerID > j.lengthExtension) && data["Linkedid"] == data["Uniqueid"] && data["ChannelState"] == "6" {

		if callerID > j.lengthExtension {
			called = Normalization(data["CallerIDNum"])
			calling = data["ConnectedLineNum"]
		} else {
			called = Normalization(data["ConnectedLineNum"])
			calling = data["CallerIDNum"]
		}
		var jsonData = j.OutputByte("{\"ID\": \"%s\",\"STS\": \"2\" ,\"SRC\": \"%s\",\"DAKHELI\": \"%s\"}", data["Linkedid"], called, calling)
		var headers = map[string]string{"accept": "*/*", "Content-Type": "application/json", "Access-token": j.token}
		jolRestApi.SendRequestJSON(j.urlAnswer, jsonData, headers)
	}
}

func (j *jolAmi) BridgeEnterHandle(data map[string]string) {
	j.DebugCheck("BridgeEnterHandle: {{ %s }}\n", data)

	// ***
	uniqueid, ok := GetUniqueID(data)
	if !ok {
		return
	}
	if loaded, ok := j.callStore.Load(uniqueid); ok && data["ChannelState"] == "6" {
		call := loaded.(*ChannelInfo)
		j.DebugCheck("Link: %s => ConnectedLineNum:%s, CallerIDNum:%s, Exten:%s \n", call.LinkedID, data["ConnectedLineNum"], data["CallerIDNum"], data["Exten"])
		call.Callee = data["ConnectedLineNum"]

		j.callStore.Store(data["Uniqueid"], call)

		var jsonData = j.OutputByte("{\"ID\": \"%s\",\"STS\": \"1\" ,\"SRC\": \"%s\",\"DAKHELI\": \"%s\"}", uniqueid, call.Caller, call.Callee)
		var headers = map[string]string{"accept": "*/*", "Content-Type": "application/json", "Access-token": j.token}
		jolRestApi.SendRequestJSON(j.urlAnswer, jsonData, headers)
	}

	// ***
	/*
		callerID := len(data["CallerIDNum"])
		var called, calling string
		if (callerID == j.lengthExtension && len(data["ConnectedLineNum"]) > j.lengthExtension || len(data["DestExten"]) == j.lengthExtension && callerID > j.lengthExtension) && data["Linkedid"] == data["Uniqueid"] && data["ChannelState"] == "6" {

			if callerID > j.lengthExtension {
				called = Normalization(data["CallerIDNum"])
				calling = data["DestExten"]
				data["CallerIDNum"] = called
			} else {
				called = Normalization(data["DestExten"])
				calling = data["CallerIDNum"]
				data["ConnectedLineNum"] = called
			}
			var jsonData = j.OutputByte("{\"ID\": \"%s\",\"STS\": \"1\" ,\"SRC\": \"%s\",\"DAKHELI\": \"%s\"}", data["Linkedid"], called, calling)
	*/
	// var headers = map[string]string{"accept": "*/*", "Content-Type": "application/json", "Access-token": j.token}
	//jolRestApi.SendRequestJSON(j.urlAnswer, jsonData, headers)
	// }

}
func (j *jolAmi) HangupHandle(data map[string]string) {
	j.DebugCheck("HangupHandle: {{ %s }}\n", data)
	callerID := len(data["CallerIDNum"])
	var called, DAKHELI string
	if (callerID == j.lengthExtension && len(data["ConnectedLineNum"]) > j.lengthExtension || len(data["ConnectedLineNum"]) == j.lengthExtension && callerID > j.lengthExtension) && data["Linkedid"] == data["Uniqueid"] && data["ChannelState"] == "6" {

		if callerID > j.lengthExtension {
			called = Normalization(data["CallerIDNum"])
			DAKHELI = data["ConnectedLineNum"]
			data["CallerIDNum"] = called
		} else {
			called = Normalization(data["ConnectedLineNum"])
			DAKHELI = data["CallerIDNum"]
			data["ConnectedLineNum"] = called
		}
		var jsonData = j.OutputByte("{\"uniqueId\": \"%s\",\"callerId\": \"%s\",\"DAKHELI\": \"%s\"}", data["Linkedid"], called, DAKHELI)
		var headers = map[string]string{"accept": "*/*", "Content-Type": "application/json", "Access-token": j.token}
		jolRestApi.SendRequestJSON(j.urlReject, jsonData, headers)
	}
}

func (j *jolAmi) EndCallHandle(data map[string]string) {
	j.DebugCheck("CDR: {{ %s }}\n", data)
	uniqueId, ok := GetUniqueID(data)
	if !ok {
		return
	}
	// **

	if loaded, ok := j.callStore.Load(uniqueId); ok {
		call := loaded.(*ChannelInfo)
		STS := 3
		duration := "0"
		if data["Disposition"] == "ANSWERED" {
			STS = 2
			duration = data["Duration"]
		}
		var jsonData = j.OutputByte("{\"ID\": \"%s\",\"STS\":\"%d\",\"SRC\": \"%s\",\"DAKHELI\": \"%s\",\"DELAY_TIME\":\"%s\",\"fileUrl\":\"%s\"}", call.LinkedID, STS, call.Caller, call.Callee, duration, "")
		var headers = map[string]string{"accept": "*/*", "Content-Type": "application/json", "Access-token": j.token}
		j.callStore.Delete(uniqueId)
		jolRestApi.SendRequestJSON(j.urlEndCall, jsonData, headers)
	}
	// **
	/*
		callerID := len(data["Source"])
		var callerId, DAKHELI string
		if callerID == length || callerID > length {
			destination := getExtensionFromChannelAndReturn(data["DestinationChannel"], length, debug)
			if callerID > length {
				callerId = Normalization(data["Source"])

				if len(destination) == length {
					DAKHELI = destination
				} else {
					DAKHELI = data["CallerIDNum"]
				}

			} else {
				callerId = Normalization(data["Destination"])
				DAKHELI = data["CallerIDNum"]
			}

			STS := 3
			if data["Disposition"] == "ANSWERED" {
				STS = 2
			}
			var jsonData = OutputByte(debug, "{\"uniqueId\": \"%s\",\"STS\":\"%d\",\"callerId\": \"%s\",\"DAKHELI\": \"%s\",\"duration\":\"%s\",\"buildSeconds\":\"%s\",\"callStatus\":\"%s\",\"fileUrl\":\"%s\"}", data["UniqueID"], STS, callerId, DAKHELI, data["Duration"], data["BillableSeconds"], data["Disposition"], "")
	*/
	//	var headers = map[string]string{"accept": "*/*", "Content-Type": "application/json", "Access-token": token}
	//	jolRestApi.SendRequestJSON(urlAddr, jsonData, headers)

	//}
}

func (j *jolAmi) NewchannelHandle(data map[string]string) {
	j.DebugCheck("Newchanne: {{ %s }}\n", data)
	uniqueid, ok := GetUniqueID(data)
	if !ok {
		return
	}
	if uniqueid == data["Linkedid"] && data["Exten"] != "s" && (len(data["CallerIDNum"]) != len(data["Exten"])) {
		j.callStore.Store(data["Uniqueid"], &ChannelInfo{
			callerId: len(data["CallerIDNum"]),
			Caller:   data["CallerIDNum"],
			LinkedID: uniqueid,
		})
	}
}
func Normalization(callerID string) (normalizedNumber string) {

	normalizedNumber = callerID
	regRemovePlus98 := regexp.MustCompile(`^\+98\d`)
	if regRemovePlus98.MatchString(normalizedNumber) {
		normalizedNumber = normalizedNumber[3:]
	}
	regRemove98 := regexp.MustCompile(`^98\d`)
	if regRemove98.MatchString(normalizedNumber) {
		normalizedNumber = normalizedNumber[2:]
	}
	regAddZiro := regexp.MustCompile(`^\d{10}$`)
	if regAddZiro.MatchString(normalizedNumber) {
		normalizedNumber = "0" + normalizedNumber
	}
	regRemoveEnd11 := regexp.MustCompile(`^\d{12}\d*$`)
	if regRemoveEnd11.MatchString(normalizedNumber) {
		normalizedNumber = normalizedNumber[len(normalizedNumber)-11 : len(normalizedNumber)]
	}
	regLocal := regexp.MustCompile(`^\d{9}$`)
	if regLocal.MatchString(normalizedNumber) {
		normalizedNumber = normalizedNumber[1:]
	}

	return
}

func (j *jolAmi) OutputByte(format string, a ...any) []byte {
	x := fmt.Sprintf(format, a...)
	if j.debug {
		fmt.Println(x)
	}
	return []byte(x)
}

func (j *jolAmi) DebugCheck(format string, a ...any) {
	x := fmt.Sprintf(format, a...)
	if j.debug {
		fmt.Println(x)
	}
}

func (j *jolAmi) getExtensionFromChannelAndReturn(channel string) string {
	r, _ := regexp.Compile(fmt.Sprintf("^(SIP|sip|Local)/(\\d{%d})", j.lengthExtension))
	extension := r.FindStringSubmatch(channel)
	if len(extension) == 0 {
		return ""
	}

	j.DebugCheck("PeerCheck: {{ %s:%s }}\n", extension, channel)
	return extension[2]

}
func GetUniqueID(data map[string]string) (string, bool) {
	var uniqueId string
	uniqueId, ok := data["Uniqueid"]
	if !ok {
		uniqueId, ok = data["UniqueID"]
		if !ok {
			return "", false
		}
	}
	return uniqueId, true
}
