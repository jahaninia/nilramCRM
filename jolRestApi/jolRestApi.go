package jolRestApi

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

func SendRequestJSON(urlAddr string, data []byte, headers map[string]string) {
	defer func() {
		r := recover()
		if r != nil {
			fmt.Printf("%v", r)
		}
	}()

	fmt.Println("HTTP JSON POST URL:", urlAddr)

	request, error := http.NewRequest("POST", urlAddr, bytes.NewBuffer(data))

	for k, v := range headers {
		request.Header.Set(k, v)
	}
	client := &http.Client{}
	response, error := client.Do(request)
	if error != nil {
		print(error)
	}
	defer response.Body.Close()

	fmt.Println("response Status:", response.Status)
	fmt.Println("response Headers:", response.Header)
	body, _ := ioutil.ReadAll(response.Body)
	fmt.Println("response Body:", string(body))
}
