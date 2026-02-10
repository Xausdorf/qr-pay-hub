package qrcode

type QRData struct {
	ToAccount string `json:"to_account"`
	Amount    int64  `json:"amount"`
}

type Generator interface {
	Generate(data QRData) ([]byte, error)
}
