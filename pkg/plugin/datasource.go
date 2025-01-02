package plugin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type CreatedKernelResponse struct {
	Id             string    `json:"id"`
	Name           string    `json:"name"`
	LastActivity   time.Time `json:"last_activity"`
	ExecutionState string    `json:"execution_state"`
	Connections    int       `json:"connections"`
}

func createKernel(baseUrl string, token string) (*CreatedKernelResponse, error) {

	_baseUrl, err := url.Parse(baseUrl)
	if err != nil {
		log.Printf("createKernel 1 : %v", err.Error())
		return nil, errors.New("")
	}

	kernelsParts, err := url.Parse("kernels")

	if err != nil {
		log.Printf("createKernel 2 : %v", err.Error())
		return nil, errors.New("")
	}

	requestUrl := _baseUrl
	requestUrl.Path = path.Join(_baseUrl.Path, kernelsParts.Path)

	params := url.Values{}

	if token != "" {
		params.Add("token", token)
	}

	requestUrl.RawQuery = params.Encode()

	jupyterUrl := requestUrl.String()

	log.Println(jupyterUrl)

	req, err := http.NewRequest("POST", jupyterUrl, nil)

	if err != nil {
		log.Printf("createKernel 3 : %v", err.Error())
		return nil, errors.New("")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("createKernel 4 : %v", err.Error())
		return nil, errors.New("")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("createKernel 5 : %v", err.Error())
		return nil, errors.New("")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("createKernel 6 : %v", resp.StatusCode)
		return nil, errors.New("")
	}

	var kernel CreatedKernelResponse
	jerr := json.Unmarshal(body, &kernel)
	if jerr != nil {
		log.Printf("createKernel 7 : %v", jerr.Error())
		return nil, errors.New("")
	}
	return &kernel, nil
}

func deleteKernel(baseUrl string, token string, kernelId string) error {

	_baseUrl, err := url.Parse(baseUrl)
	if err != nil {
		return errors.New("")
	}

	kernelsParts, err := url.Parse(fmt.Sprintf("kernels/%s", kernelId))

	if err != nil {
		return errors.New("")
	}

	requestUrl := _baseUrl
	requestUrl.Path = path.Join(_baseUrl.Path, kernelsParts.Path)

	params := url.Values{}

	if token != "" {
		params.Add("token", token)
	}

	requestUrl.RawQuery = params.Encode()

	jupyterUrl := requestUrl.String()
	req, err := http.NewRequest("DELETE", jupyterUrl, nil)

	if err != nil {
		return errors.New("")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("")
	}

	return nil
}

// ExecuteRequest はコード実行リクエストの構造
type ExecuteRequest struct {
	Header struct {
		MsgID    string `json:"msg_id"`
		Username string `json:"username"`
		Session  string `json:"session"`
		MsgType  string `json:"msg_type"`
		Version  string `json:"version"`
	} `json:"header"`
	ParentHeader map[string]interface{} `json:"parent_header"`
	Metadata     map[string]interface{} `json:"metadata"`
	Content      struct {
		Code   string `json:"code"`
		Silent bool   `json:"silent"`
	} `json:"content"`
}

// WebSocket メッセージを受信した際の処理
func handleMessages(conn *websocket.Conn) (*[]byte, *[]byte, error) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			return nil, nil, err
		}

		// メッセージのパース
		var data map[string]interface{}
		err = json.Unmarshal(message, &data)
		if err != nil {
			log.Println("Error parsing message:", err)
			continue
		}

		// メッセージタイプによる処理
		msgType, ok := data["msg_type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "stream":
			content := data["content"].(map[string]interface{})
			log.Println("Output:", content["text"])
		case "execute_reply":
			log.Println("Execution completed.")
			return nil, nil, nil
		case "execute_result":
			content := data["content"].(map[string]interface{})
			d := content["data"].(map[string]interface{})
			result := d["text/plain"].(string)
			log.Println(result)

			base64Text := result[1 : len(result)-1]
			log.Println(base64Text)

			base64List := strings.Split(base64Text, ".")

			decoded1, err := base64.StdEncoding.DecodeString(base64List[0])
			if err != nil {
				fmt.Println("Error decoding Base64:", err)
				return nil, nil, fmt.Errorf("error decoding Base64: %v", err.Error())
			}

			log.Println(string(decoded1))

			decoded2, err := base64.StdEncoding.DecodeString(base64List[1])
			if err != nil {
				fmt.Println("Error decoding Base64:", err)
				return nil, nil, fmt.Errorf("error decoding Base64: %v", err.Error())
			}

			log.Println(string(decoded2))

			return &decoded1, &decoded2, nil

		case "error":
			content := data["content"].(map[string]interface{})
			log.Println("Error:", content["traceback"])
			return nil, nil, fmt.Errorf("%+v", content["traceback"])
		}
	}
}

func doPython(baseUrl string, token string, kernelId string, code string, resultCode string, timeNameListCode string) (*map[string][]interface{}, *[]string, error) {
	// Jupyterサーバー設定
	wsBaseUrl := strings.ReplaceAll(baseUrl, "http", "ws")
	sessionID := uuid.New().String() // 一意のセッションID

	wsBase, err := url.Parse(wsBaseUrl)
	if err != nil {
		log.Printf("doPython 1 : %v", err.Error())
		return nil, nil, err
	}

	channelsPart, err := url.Parse(fmt.Sprintf("kernels/%s/channels", kernelId))

	if err != nil {
		log.Printf("doPython 2 : %v", err.Error())
		return nil, nil, err
	}

	channelsUrl := wsBase
	channelsUrl.Path = path.Join(wsBase.Path, channelsPart.Path)

	params := url.Values{}

	if token != "" {
		params.Add("token", token)
	}

	channelsUrl.RawQuery = params.Encode()

	// WebSocket接続を確立
	conn, _, err := websocket.DefaultDialer.Dial(channelsUrl.String(), nil)
	if err != nil {
		log.Printf("doPython 3 : %v", err.Error())
		log.Fatal("Failed to connect to WebSocket:", err)
		return nil, nil, err
	}
	defer conn.Close()

	{
		// コード実行リクエストを作成
		request := ExecuteRequest{}
		request.Header.MsgID = uuid.New().String()
		request.Header.Username = "user"
		request.Header.Session = sessionID
		request.Header.MsgType = "execute_request"
		request.Header.Version = "5.3"
		request.Content.Code = code // 実行するコード
		request.Content.Silent = false

		// JSONにシリアライズして送信
		message, err := json.Marshal(request)
		if err != nil {
			log.Printf("doPython 4 : %v", err.Error())
			log.Fatal("Failed to marshal request:", err)
			return nil, nil, err
		}
		err = conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("doPython 5 : %v", err.Error())
			log.Fatal("Failed to send message:", err)
			return nil, nil, err
		}

		log.Println("Code execution request sent.")

		// メッセージを受信
		handleMessages(conn)
	}

	{
		// コード実行リクエストを作成
		request := ExecuteRequest{}
		request.Header.MsgID = uuid.New().String()
		request.Header.Username = "user"
		request.Header.Session = sessionID
		request.Header.MsgType = "execute_request"
		request.Header.Version = "5.3"
		request.Content.Code = fmt.Sprintf(`import json
import base64
base64.b64encode(json.dumps(%s,ensure_ascii=True,indent=None).encode("utf-8")).decode('ascii') \
+ "." \
+ base64.b64encode(json.dumps((%s if "%s" in globals() else []),ensure_ascii=True,indent=None).encode("utf-8")).decode('ascii')`, resultCode, timeNameListCode, timeNameListCode) // 実行するコード
		request.Content.Silent = false

		// JSONにシリアライズして送信
		message, err := json.Marshal(request)
		if err != nil {
			log.Printf("doPython 6 : %v", err.Error())
			log.Fatal("Failed to marshal request:", err)
			return nil, nil, err
		}
		err = conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("doPython 7 : %v", err.Error())
			log.Fatal("Failed to send message:", err)
			return nil, nil, err
		}

		log.Println("Code execution request sent.")

		// メッセージを受信
		resultData, timeNameListData, err := handleMessages(conn)
		if err != nil {
			log.Printf("doPython 8 : %v", err.Error())
			log.Fatal("Failed to recive message:", err)
			return nil, nil, err
		}

		var jsonData map[string][]interface{}
		jerr := json.Unmarshal(*resultData, &jsonData)

		if jerr != nil {
			log.Printf("doPython 9 : %v", jerr.Error())
			log.Fatal("Failed unmarshal message:", jerr)
			return nil, nil, jerr
		}

		var timeNameListJsonData []string
		jerr2 := json.Unmarshal(*timeNameListData, &timeNameListJsonData)

		if jerr2 != nil {
			log.Printf("doPython 10 : %v", jerr2.Error())
			log.Fatal("Failed unmarshal message:", jerr2)
			return nil, nil, jerr2
		}

		return &jsonData, &timeNameListJsonData, nil
	}
}

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces - only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// NewDatasource creates a new datasource instance.
func NewDatasource(_ context.Context, _ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &Datasource{}, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct{}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	Code          string `json:"code"`
	ResultCode    string `json:"resultCode"`
	TimeNamesCode string `json:"timeNamesCode"`
}

//	type envItem struct {
//		Name  string `json:"name"`
//		Value string `json:"value"`
//	}
type settingJsonDataModel struct {
	// Env    []envItem `json:"env"`
	ApiUrl string `json:"apiBaseUrl"`
}

func contains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func (d *Datasource) query(_ context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm queryModel
	var sjdm settingJsonDataModel

	log.Println("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@")
	log.Println(string(query.JSON))
	log.Println(string(pCtx.DataSourceInstanceSettings.JSONData))
	log.Println(string(pCtx.DataSourceInstanceSettings.DecryptedSecureJSONData["apiToken"]))

	// return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", "aaa bbbb"))

	{
		err := json.Unmarshal(query.JSON, &qm)
		if err != nil {
			return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
		}
	}

	{
		err := json.Unmarshal(pCtx.DataSourceInstanceSettings.JSONData, &sjdm)
		if err != nil {
			return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
		}
	}

	baseUrl := sjdm.ApiUrl
	token := pCtx.DataSourceInstanceSettings.DecryptedSecureJSONData["apiToken"]

	kernel, err := createKernel(baseUrl, token)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("create kernel: %v", err.Error()))
	}

	log.Printf("%+v", *kernel)

	resultData, timeNameList, _ := doPython(baseUrl, token, kernel.Id, qm.Code, qm.ResultCode, qm.TimeNamesCode)

	dk_err := deleteKernel(baseUrl, token, kernel.Id)
	if dk_err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("delete kernel: %v", dk_err.Error()))
	}

	// create data frame response.
	// For an overview on data frames and how grafana handles them:
	// https://grafana.com/developers/plugin-tools/introduction/data-frames
	frame := data.NewFrame("response")

	// add fields.
	// frame.Fields = append(frame.Fields,
	// 	// data.NewField("time", nil, []time.Time{query.TimeRange.From, query.TimeRange.To}),
	// 	data.NewField("x", nil, []int64{10, 50}),
	// 	data.NewField("values", nil, []int64{10, 50}),
	// )

	log.Println("****************************************************")

	if resultData != nil {

		for key, values := range *resultData {
			log.Printf("key = %s", key)

			if contains(*timeNameList, key) {
				newValues := make([]time.Time, len(values))
				for i, v := range values {
					if val, ok := v.(string); ok {
						log.Printf("time format: %s", val)
						timeVal, err := time.Parse(time.RFC3339, val)
						if err != nil {
							log.Printf("time format converted: %v", err)
						} else {
							log.Printf("time format converted: %v", timeVal)
							newValues[i] = timeVal
						}
					} else {
						log.Fatalf("unsupported value type: %T", v)
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(key, nil, newValues))
				continue
			}
			fmt.Print()
			first := values[0]

			switch first.(type) {
			case int:
				newValues := make([]int, len(values))
				for i, v := range values {
					if val, ok := v.(int); ok {
						newValues[i] = val
					} else {
						log.Fatalf("unsupported value type: %T", v)
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(key, nil, newValues))
			case float64:
				newValues := make([]float64, len(values))
				for i, v := range values {
					if val, ok := v.(float64); ok {
						newValues[i] = val
					} else {
						log.Fatalf("unsupported value type: %T", v)
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(key, nil, newValues))

			case string:
				newValues := make([]string, len(values))
				for i, v := range values {
					if val, ok := v.(string); ok {
						newValues[i] = val
					} else {
						log.Fatalf("unsupported value type: %T", v)
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(key, nil, newValues))

			}

		}
	}

	// add the frames to the response.
	response.Frames = append(response.Frames, frame)

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// res := &backend.CheckHealthResult{}
	// config, err := models.LoadPluginSettings(*req.PluginContext.DataSourceInstanceSettings)

	// if err != nil {
	// 	res.Status = backend.HealthStatusError
	// 	res.Message = "Unable to load settings"
	// 	return res, nil
	// }

	// if config.Secrets.ApiKey == "" {
	// 	res.Status = backend.HealthStatusError
	// 	res.Message = "API key is missing"
	// 	return res, nil
	// }

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working",
	}, nil
}
