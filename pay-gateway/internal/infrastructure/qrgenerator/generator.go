package qrgenerator

import (
	"encoding/json"

	qr "github.com/skip2/go-qrcode"

	"github.com/Xausdorf/qr-pay-hub/pay-gateway/internal/domain/qrcode"
)

type Generator struct {
	size int
}

func NewGenerator(size int) *Generator {
	return &Generator{size: size}
}

func (g *Generator) Generate(data qrcode.QRData) ([]byte, error) {
	content, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return qr.Encode(string(content), qr.Medium, g.size)
}
