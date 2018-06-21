package fresh

import (
	"bytes"
	"regexp"
	"time"
)

// RequestHeader 请求头
type RequestHeader struct {
	IfModifiedSince []byte
	IfNoneMatch     []byte
	CacheControl    []byte
}

// ResponseHeader 响应头
type ResponseHeader struct {
	ETag         []byte
	LastModified []byte
}

var noCacheReg = regexp.MustCompile(`(?:^|,)\s*?no-cache\s*?(?:,|$)`)

var weekTagPrefix = []byte("W/")

func parseTokenList(buf []byte) [][]byte {
	end := 0
	start := 0
	count := len(buf)
	list := make([][]byte, 0)
	for index := 0; index < count; index++ {
		switch int(buf[index]) {
		// 空格
		case 0x20:
			if start == end {
				end = index + 1
				start = end
			}
		// , 号
		case 0x2c:
			list = append(list, buf[start:end])
			end = index + 1
			start = end
		default:
			end = index + 1
		}
	}
	list = append(list, buf[start:end])
	return list
}

func parseHTTPDate(date string) int64 {
	t, err := time.Parse(time.RFC1123, date)
	if err != nil {
		return 0
	}
	return t.Unix()
}

// Check 判断响应是否fresh
func Check(modifiedSince, noneMatch, cacheControl, lastModified, etag []byte) bool {
	if len(modifiedSince) == 0 && len(noneMatch) == 0 {
		return false
	}
	if len(cacheControl) != 0 && noCacheReg.Match(cacheControl) {
		return false
	}
	// if none match
	if len(noneMatch) != 0 && (len(noneMatch) != 1 || noneMatch[0] != byte('*')) {
		if len(etag) == 0 {
			return false
		}
		matches := parseTokenList(noneMatch)
		etagStale := true
		for _, match := range matches {
			if bytes.Equal(match, etag) {
				etagStale = false
				break
			}
			if bytes.HasPrefix(match, weekTagPrefix) && bytes.Equal(match[2:], etag) {
				etagStale = false
				break
			}
			if bytes.HasPrefix(etag, weekTagPrefix) && bytes.Equal(etag[2:], match) {
				etagStale = false
				break
			}
		}
		if etagStale {
			return false
		}
	}
	// if modified since
	if len(modifiedSince) != 0 {
		if len(lastModified) == 0 {
			return false
		}
		lastModifiedUnix := parseHTTPDate(string(lastModified))
		modifiedSinceUnix := parseHTTPDate(string(modifiedSince))
		if lastModifiedUnix == 0 || modifiedSinceUnix == 0 {
			return false
		}
		if modifiedSinceUnix < lastModifiedUnix {
			return false
		}
	}
	return true
}

// Fresh 判断该请求是否 fresh
func Fresh(reqHeader *RequestHeader, resHeader *ResponseHeader) bool {
	return Check(reqHeader.IfModifiedSince, reqHeader.IfNoneMatch, reqHeader.CacheControl, resHeader.LastModified, resHeader.ETag)
}
