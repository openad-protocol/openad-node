package common

const (
	SUCCESS       = 0
	HASHLENERROR  = 1
	HASHDATAERROR = 2
	SERVERERROR   = 3
)

var CodeMessageMap = map[int32]string{
	SUCCESS:       "success",
	HASHLENERROR:  "hash use sha256. match 32 bytes",
	HASHDATAERROR: "hash error",
	SERVERERROR:   "verify server get error from layer2 server",
}

type VerifyResponse struct {
	Code    int32       `json:"code"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}
