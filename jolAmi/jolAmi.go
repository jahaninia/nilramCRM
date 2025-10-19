package jolAmi

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/ivahaev/amigo"
	"jahaninia.ir/cloud/asterisk/pbx/jolRestApi"
)

type ChannelInfo struct {
	CallerId  int
	Caller    string
	Callee    string
	SRC       string
	Extension string
	LinkedID  string
	Direction int
	STS       int
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
	//case "Newstate":
	//j.NewstateHandle(e)
	case "BridgeEnter":
		j.BridgeEnterHandle(e)
	// case "BridgeLeave":
	// 	BridgeLeaveHandle(e, config.UrlReject, config.Token, config.Length, config.Debug)
	case "NewConnectedLine":
		j.NewConnectedLineHandle(e)
	//case "Hangup":
	//	j.HangupHandle(e)
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
	if loaded, ok := j.callStore.Load(data["Linkedid"]); ok && data["DestExten"] != "s" {
		call := loaded.(*ChannelInfo)
		fmt.Printf("**************Call Info in memory:%#v\n", call)

		j.DebugCheck("Link: %s => DestCallerIDNum:%s, DestExten:%s, DialString:%s \n", call.LinkedID, data["DestCallerIDNum"], data["DestExten"], data["DialString"])

		if call.Direction == 0 && call.Callee == "NULL" && call.LinkedID == uniqueid {
			if data["Context"] == "macro-dial-one" {
				call.Callee = data["DialString"]
				call.Extension = call.Callee
			}
		} else if call.Direction == 0 && call.Callee == "NULL" && call.LinkedID != uniqueid {
			if data["Context"] == "from-queue" {
				call.Callee = data["Exten"]
				call.Extension = call.Callee
			} else {

				call.Callee = data["DestExten"]
				call.Extension = call.Callee
			}

		}
		// if data["Context"] == "macro-dialout-trunk" {
		// }else if data["Context"] == "ext-queues" {
		// 	call.Callee = data["DestExten"]
		// } else {
		// 	call.Callee = data["DialString"]
		// }
		j.callStore.Store(call.LinkedID, call)

		var jsonData = j.OutputByte("{\"ID\": \"%s\",\"STS\": \"%d\" ,\"SRC\": \"%s\",\"DAKHELI\": \"%s\",\"IN_OUT\": \"%d\"}", call.LinkedID, call.STS, call.SRC, call.Extension, call.Direction)
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
func (j *jolAmi) NewConnectedLineHandle(data map[string]string) {
	j.DebugCheck("NewConnectedLineHandle: {{ %s }}\n", data)
	return
	uniqueid, ok := GetUniqueID(data)
	if !ok {
		return
	}
	if loaded, ok := j.callStore.Load(data["Linkedid"]); ok {
		call := loaded.(*ChannelInfo)
		/*^@CallerIDName:ms-ziari
		  ^@CallerIDNum:211
		  ^@Channel:SIP/211-0000a3b9
		  ^@ChannelState:0
		  ^@ChannelStateDesc:Down
		  ^@ConnectedLineName:+982184373000
		  ^@ConnectedLineNum:+982184373000
		  ^@Context:from-internal
		  ^@Event:NewConnectedLine
		  ^@Exten:211
		  ^@Language:pr
		  ^@Linkedid:1759898774.928073
		  ^@Priority:1
		  ^@Privilege:call,all
		  ^@TimeReceived:2025-10-08T08:16:30.309225948+03:30
		  ^@Uniqueid:1759898790.928076]
		*/

		j.DebugCheck("Link: %s => DestCallerIDNum:%s, DestExten:%s, DialString:%s \n", call.LinkedID, data["DestCallerIDNum"], data["DestExten"], data["DialString"])
		if call.Direction == 0 && call.Callee == "NULL" && call.LinkedID != uniqueid && data["ChannelState"] == "4" {
			if data["Context"] == "from-queue" {
				call.Callee = data["Exten"]
				call.Extension = call.Callee
			}

			call.Callee = data["DestExten"]
			call.Extension = call.Callee

		}
		// if data["Context"] == "macro-dialout-trunk" {
		// }else if data["Context"] == "ext-queues" {
		// 	call.Callee = data["DestExten"]
		// } else {
		// 	call.Callee = data["DialString"]
		// }
		j.callStore.Store(call.LinkedID, call)

		var jsonData = j.OutputByte("{\"ID\": \"%s\",\"STS\": \"%d\" ,\"SRC\": \"%s\",\"DAKHELI\": \"%s\",\"IN_OUT\": \"%d\"}", call.LinkedID, call.STS, call.SRC, call.Extension, call.Direction)
		var headers = map[string]string{"accept": "*/*", "Content-Type": "application/json", "Access-token": j.token}
		jolRestApi.SendRequestJSON(j.urlCall, jsonData, headers)
	}

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

	uniqueid, ok := GetUniqueID(data)
	if !ok {
		return
	}

	// ***

	if loaded, ok := j.callStore.Load(data["Linkedid"]); ok && data["ChannelState"] == "6" && data["Linkedid"] == uniqueid {
		call := loaded.(*ChannelInfo)
		call.STS = 2
		j.callStore.Store(call.LinkedID, call)

		var jsonData = j.OutputByte("{\"ID\": \"%s\",\"STS\": \"%d\" ,\"SRC\": \"%s\",\"DAKHELI\": \"%s\"}", call.LinkedID, call.STS, call.SRC, call.Extension)
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

	// ***
	uniqueid, ok := GetUniqueID(data)
	if !ok {
		return
	}

	if loaded, ok := j.callStore.Load(data["Linkedid"]); ok && data["Exten"] == "h" {
		call := loaded.(*ChannelInfo)
		if call.Direction == 0 && call.LinkedID == uniqueid {
			call.Callee = data["ConnectedLineNum"]
			call.Extension = call.Callee

		}
		j.callStore.Store(call.LinkedID, call)

		var jsonData = j.OutputByte("{\"ID\": \"%s\",\"STS\": \"%d\" ,\"SRC\": \"%s\",\"DAKHELI\": \"%s\",\"IN_OUT\": \"%d\"}", call.LinkedID, call.STS, call.SRC, call.Extension, call.Direction)
		var headers = map[string]string{"accept": "*/*", "Content-Type": "application/json", "Access-token": j.token}
		jolRestApi.SendRequestJSON(j.urlReject, jsonData, headers)

		// ***
		/*
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
				var jsonData = j.OutputByte("{\"uniqueId\": \"%s\",\"callerId\": \"%s\",\"DAKHELI\": \"%s\"}", data["Linkedid"], called, DAKHELI)*/
		//var headers = map[string]string{"accept": "*/*", "Content-Type": "application/json", "Access-token": j.token}
		//jolRestApi.SendRequestJSON(j.urlReject, jsonData, headers)

	}
}

func (j *jolAmi) EndCallHandle(data map[string]string) {
	j.DebugCheck("CDR: {{ %s }}\n", data)
	uniqueid, ok := GetUniqueID(data)
	if !ok {
		return
	}
	// **

	if loaded, ok := j.callStore.Load(uniqueid); ok {
		call := loaded.(*ChannelInfo)
		fmt.Printf("**************CDR**Call Info in memory:%#v\n", call)
		duration := "0"
		if call.STS != 2 {
			call.STS = 3
		}

		duration = data["Duration"]

		var jsonData = j.OutputByte("{\"ID\": \"%s\",\"STS\":\"%d\",\"SRC\": \"%s\",\"DAKHELI\": \"%s\",\"DELAY_TIME\":\"%s\",\"fileUrl\":\"%s\"}", call.LinkedID, call.STS, call.SRC, call.Extension, duration, "")
		var headers = map[string]string{"accept": "*/*", "Content-Type": "application/json", "Access-token": j.token}
		j.callStore.Delete(call.LinkedID)
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
	j.DebugCheck("Newchannel: {{ %s }}\n", data)
	uniqueid, ok := GetUniqueID(data)
	if !ok {
		return
	}

	if loaded, ok := j.callStore.Load(data["Linkedid"]); ok {
		call := loaded.(*ChannelInfo)
		fmt.Printf("**************Newchanne**Call Info in memory:%#v\n", call)

		if data["Context"] == "from - queue" && call.Direction == 0 {
			call.Callee = data["Exten"]
			call.Extension = call.Callee
			j.callStore.Store(call.LinkedID, call)
		}
		if data["Context"] == "from-internal" && len(data["CallerIDNum"]) == j.lengthExtension && data["ChannelState"] == "0" {
			call.Callee = data["CallerIDNum"]
			call.Extension = call.Callee
			j.callStore.Store(call.LinkedID, call)

		}
	} else if uniqueid == data["Linkedid"] && data["Exten"] != "s" {
		var channel ChannelInfo
		channel.Direction = 0
		channel.LinkedID = uniqueid
		channel.CallerId = len(data["CallerIDNum"])
		channel.STS = 1

		if channel.CallerId > j.lengthExtension && data["Context"] == "from-trunk" {
			channel.Caller = Normalization(data["CallerIDNum"])
			channel.SRC = channel.Caller
			channel.Callee = "NULL"
		}

		if data["Context"] == "from-internal" && data["ChannelState"] == "0" {
			channel.Direction = 1
			channel.Callee = data["Exten"][1:]
			channel.Caller = data["CallerIDNum"]
			channel.Extension = channel.Caller
			channel.SRC = channel.Callee
		}
		fmt.Printf("**************Newchanne*END*Call Info in memory:%#v\n", channel)

		j.callStore.Store(data["Uniqueid"], &channel)
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
