package servicemesh

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/golang/protobuf/proto"
)

func apiRequest(serverUrl *url.URL, endpoint string, req proto.Message, resp proto.Message) error {
	httpRsp, err := post(context.TODO(), endpointNameToPublicAPIURL(serverUrl, endpoint), req)
	if err != nil {
		return fmt.Errorf("post request failed: %s", err.Error())
	}
	defer httpRsp.Body.Close()

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
