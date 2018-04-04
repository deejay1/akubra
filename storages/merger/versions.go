package merger

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/allegro/akubra/log"
	"github.com/allegro/akubra/storages/merger/s3datatypes"
	"github.com/allegro/akubra/transport"
)

// MergeVersionsResponses unifies responses from multiple backends
func MergeVersionsResponses(successes []transport.ResErrTuple) (resp *http.Response, err error) {
	if len(successes) == 0 {
		log.Printf("No successful response")
		err = fmt.Errorf("No successful responses")
		return
	}
	keysContainer := &objectsContainer{
		list: make([]fmt.Stringer, 0),
		set:  make(map[string]struct{}),
	}
	var listBucketResult s3datatypes.ListVersionsResult
	for _, tuple := range successes {
		resp = tuple.Res
		listBucketResult = extractListVersionsResults(resp)
		keysContainer.append(listBucketResult.Version.ToStringer()...)
		keysContainer.append(listBucketResult.DeleteMarker.ToStringer()...)
	}

	req := successes[0].Res.Request
	reqQuery := req.URL.Query()
	maxKeysQuery := reqQuery.Get("max-keys")
	maxKeys, err := strconv.Atoi(maxKeysQuery)
	if err != nil {
		maxKeys = 1000
	}

	listBucketResult = createVersionResultSet(keysContainer, maxKeys, listBucketResult)

	bodyBytes, err := xml.Marshal(listBucketResult)
	if err != nil {
		log.Debug("Problem marshalling ObjectStore response body, %s", err)
		return nil, err
	}
	buf := bytes.NewBuffer(bodyBytes)
	resp.Body = ioutil.NopCloser(buf)
	resp.ContentLength = int64(buf.Len())
	resp.Header = http.Header{}
	resp.Header.Set("content-length", fmt.Sprintf("%d", buf.Len()))
	resp.Header.Set("content-type", "application/xml")
	return resp, nil
}

func extractListVersionsResults(resp *http.Response) s3datatypes.ListVersionsResult {
	lbr := s3datatypes.ListVersionsResult{}
	if resp.Body == nil {
		return lbr
	}

	buf := &bytes.Buffer{}
	if _, rerr := buf.ReadFrom(resp.Body); rerr != nil {
		log.Debugf("Problem reading ObjectStore response body, %s", rerr)
		return lbr
	}

	if cerr := resp.Body.Close(); cerr != nil {
		log.Debugf("Problem closing ObjectStore response body, %s", cerr)
		return lbr
	}

	bodyBytes := buf.Bytes()
	err := xml.Unmarshal(bodyBytes, &lbr)
	if err != nil {
		log.Debugf("ListBucketResult unmarshalling problem %s", err)
	}

	return lbr
}

func createVersionResultSet(keysContainer *objectsContainer, maxKeys int, versionsResult s3datatypes.ListVersionsResult) s3datatypes.ListVersionsResult {
	deleteMarkers := s3datatypes.DeleteMarkerInfos{}
	versions := s3datatypes.VersionInfos{}
	keys := keysContainer.first(maxKeys + 1)
	var lastMarker s3datatypes.VersionMarker
	for _, key := range keys[0:maxKeys] {
		switch v := key.(type) {
		case s3datatypes.DeleteMarkerInfo:
			deleteMarkers = append(deleteMarkers, v)
			lastMarker = v
		case s3datatypes.VersionInfo:
			versions = append(versions, v)
			lastMarker = v
		}

	}
	versionsResult.Version = versions
	versionsResult.DeleteMarker = deleteMarkers

	versionsResult.KeyMarker = lastMarker.GetKey()
	versionsResult.VersionIDMarker = lastMarker.GetVersionID()

	versionsResult.IsTruncated = keysContainer.Len() > maxKeys
	if versionsResult.IsTruncated {
		nextMarker := keys[maxKeys+1].(s3datatypes.VersionMarker)
		versionsResult.NextKeyMarker = nextMarker.GetKey()
		versionsResult.NextVersionIDMarker = nextMarker.GetVersionID()
	}

	return versionsResult
}
