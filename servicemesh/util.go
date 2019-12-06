package servicemesh

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/golang/protobuf/proto"

	pb "github.com/zdnscloud/cluster-agent/servicemesh/public"
)

const ErrorHeader = "linkerd-error"

func apiRequest(serverUrl *url.URL, endpoint string, req proto.Message, resp proto.Message) error {
	httpRsp, err := post(context.TODO(), endpointNameToPublicAPIURL(serverUrl, endpoint), req)
	if err != nil {
		return fmt.Errorf("post request failed: %s", err.Error())
	}
	defer httpRsp.Body.Close()

	if err := checkIfResponseHasError(httpRsp); err != nil {
		return err
	}

	reader := bufio.NewReader(httpRsp.Body)
	return fromByteStreamToProtocolBuffers(reader, resp)
}

func endpointNameToPublicAPIURL(serverUrl *url.URL, endpoint string) *url.URL {
	return serverUrl.ResolveReference(&url.URL{Path: endpoint})
}

func post(ctx context.Context, url *url.URL, req proto.Message) (*http.Response, error) {
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %s", err.Error())
	}

	httpReq, err := http.NewRequest(http.MethodPost, url.String(), bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("new http request failed: %s", err.Error())
	}

	return http.DefaultClient.Do(httpReq.WithContext(ctx))
}

func checkIfResponseHasError(rsp *http.Response) error {
	errorMsg := rsp.Header.Get(ErrorHeader)
	if errorMsg != "" {
		reader := bufio.NewReader(rsp.Body)
		var apiError pb.ApiError
		err := fromByteStreamToProtocolBuffers(reader, &apiError)
		if err != nil {
			return fmt.Errorf("response has %s header [%s], but response body didn't contain protobuf error: %v",
				ErrorHeader, errorMsg, err)
		}

		return fmt.Errorf("response get error: %s", apiError.Error)
	}

	if rsp.StatusCode != http.StatusOK {
		if rsp.Body != nil {
			bytes, err := ioutil.ReadAll(rsp.Body)
			if err == nil && len(bytes) > 0 {
				return fmt.Errorf("http error, status code [%d] (unexpected api response: %s)", rsp.StatusCode, string(bytes))
			}
		}

		return fmt.Errorf("http error, status code [%d] (unexpected api response)", rsp.StatusCode)
	}

	return nil
}

func fromByteStreamToProtocolBuffers(byteStreamContainingMessage *bufio.Reader, out proto.Message) error {
	messageAsBytes, err := deserializePayloadFromReader(byteStreamContainingMessage)
	if err != nil {
		return fmt.Errorf("error reading byte stream header: %v", err)
	}

	err = proto.Unmarshal(messageAsBytes, out)
	if err != nil {
		return fmt.Errorf("error unmarshalling array of [%d] bytes error: %v", len(messageAsBytes), err)
	}

	return nil
}

func deserializePayloadFromReader(reader *bufio.Reader) ([]byte, error) {
	messageLengthAsBytes := make([]byte, 4)
	_, err := io.ReadFull(reader, messageLengthAsBytes)
	if err != nil {
		return nil, fmt.Errorf("error while reading message length: %v", err)
	}
	messageLength := int(binary.LittleEndian.Uint32(messageLengthAsBytes))

	messageContentsAsBytes := make([]byte, messageLength)
	_, err = io.ReadFull(reader, messageContentsAsBytes)
	if err != nil {
		return nil, fmt.Errorf("error while reading bytes from message: %v", err)
	}

	return messageContentsAsBytes, nil
}
