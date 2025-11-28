package xendit

import (
	"encoding/json"
	"fmt"
	"io"
	"myGreenMarket/domain"
	"net/http"
	"strings"
)

type XenditConfig struct {
	XenditApi          string
	XenditUrl          string
	SuccessRedirectUrl string
	FailureRedirectUrl string
}

type XenditRepository struct {
	xenditConfig XenditConfig
}

func NewXenditRepository(cfg XenditConfig) *XenditRepository {
	return &XenditRepository{
		cfg,
	}
}

func (r XenditRepository) XenditInvoiceUrl(purpose, username, email, name, category string, userId, productID, quantity, paymentId int, amount, price float64) (string, error) {

	url := r.xenditConfig.XenditUrl
	method := "POST"
	var description string
	var duration int64
	switch purpose {
	case "TRANSFER":
		description = fmt.Sprintf("payment order %.2f", amount)
		duration = 3600
	case "TOPUP":
		description = fmt.Sprintf("top up wallet %.2f", amount)
		duration = 86400
	}

	payload := strings.NewReader(fmt.Sprintf(`{
		"external_id": "%d|%d|%d|%s",
		"amount": %.2f,
		"description": "%s",
		"invoice_duration": %d,
		"customer": {
			"email": "%s"
		},
		"success_redirect_url": "%s",
		"failure_redirect_url": "%s",
		"currency": "IDR",
		"items": [
			{
			"name": "%s",
			"quantity": %d,
			"price": %.2f,
			"category": "%s"
			}
		],
		"metadata": {
			"store": "MyGreenMarket"
		}
	}      `, paymentId, userId, productID, purpose, amount, description, duration, email, r.xenditConfig.SuccessRedirectUrl, r.xenditConfig.FailureRedirectUrl, name, quantity, price, category))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	// req.Header.Add("Authorization", fmt.Sprintf("Basic %s", r.xenditApi))
	req.SetBasicAuth(r.xenditConfig.XenditApi, "")
	req.Header.Add("Cookie", "__cf_bm=_y6J2eEmO2_wiPddsvXgUQS24DJdIlPIDViHq8aEa4c-1762356798.765628-1.0.1.1-5F1zRs5pVcS07hwmvinbN239JL7gVaEm_IE0ocMvmLg79mWOrcvcuVYPjuaMQLDGI49MIp3ACXcwfnbcgXrH6kN_MYkpd6p7autz.xSS8E9aKC.eqVUKb09MH69j_udx")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var xenditReponse domain.XenditResponse
	err = json.Unmarshal(body, &xenditReponse)
	if err != nil {
		return "", err
	}

	return xenditReponse.InvoiceURL, nil
}
