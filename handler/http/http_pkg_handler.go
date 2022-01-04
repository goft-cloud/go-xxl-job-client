package http

import (
	"bytes"
	"strings"

	getty "github.com/apache/dubbo-getty"
	"github.com/goft-cloud/go-xxl-job-client/v2/transport"
)

const (
	pkgSplitStr = "POST"
)

// PkgHandlerRes struct
type PkgHandlerRes struct {
	LastPkg []byte
	Valid   bool
}

// PackageHandler struct
type PackageHandler struct {
	pkgHandlerRes *PkgHandlerRes
}

func NewHttpPackageHandler() getty.ReadWriter {
	return &PackageHandler{
		pkgHandlerRes: &PkgHandlerRes{},
	}
}

func (h *PackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	var res []*transport.HttpRequestPkg
	length := len(data)

	if h.pkgHandlerRes.Valid { // 粘包
		var buffer bytes.Buffer
		buffer.Write(h.pkgHandlerRes.LastPkg)
		buffer.Write(data)
		data = buffer.Bytes()
		h.pkgHandlerRes.Valid = false
		h.pkgHandlerRes.LastPkg = nil
	}

	str := string(data[:]) // 需要分包
	strs := strings.Split(str, pkgSplitStr)
	if len(strs) > 0 {
		for _, s := range strs {
			if s != "" {
				req, isFinsh, remind := transport.ParseHttpRequestPkg(s)
				if isFinsh {
					res = append(res, req)
				} else {
					h.unpacks(remind)
				}
			}
		}
	}

	return res, length, nil
}

func (h *PackageHandler) Write(ss getty.Session, p interface{}) ([]byte, error) {
	pkg := p.(*transport.HttpResponsePkg)
	return pkg.Decoder(), nil
}

func (h *PackageHandler) unpacks(s []byte) {
	var buffer bytes.Buffer
	buffer.WriteString("POST")
	buffer.Write(s)
	h.pkgHandlerRes.Valid = true
	h.pkgHandlerRes.LastPkg = buffer.Bytes()
}
