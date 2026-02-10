package generateqr

import (
	"github.com/Xausdorf/qr-pay-hub/pay-gateway/internal/domain/qrcode"
)

type Request struct {
	AccountID string
	Amount    int64
}

type UseCase struct {
	generator qrcode.Generator
}

func NewUseCase(generator qrcode.Generator) *UseCase {
	return &UseCase{generator: generator}
}

func (uc *UseCase) Execute(req Request) ([]byte, error) {
	return uc.generator.Generate(qrcode.QRData{
		ToAccount: req.AccountID,
		Amount:    req.Amount,
	})
}
