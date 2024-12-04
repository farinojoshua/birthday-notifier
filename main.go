package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Youth struct {
	Nama         string
	TanggalLahir time.Time
	JenisKelamin string
	Alamat       string
	Hobi         string
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func bacaGoogleSheet(spreadsheetID, readRange string) ([]Youth, error) {
	b, err := os.ReadFile("credential.json")
	if err != nil {
		return nil, fmt.Errorf("tidak bisa membaca file credential.json: %v", err)
	}

	config, err := google.JWTConfigFromJSON(b, sheets.SpreadsheetsReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("gagal Membuat config")
	}

	client := config.Client(context.Background())

	srv, err := sheets.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("gagal membuat service: %v", err)
	}

	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("gagal membaca sheet: %v", err)
	}

	var youthList []Youth
	for _, row := range resp.Values {
		if len(row) < 5 {
			continue
		}

		nama := row[0].(string)
		rawTanggal := row[1].(string)
		tanggalNormalized := normalisasiTanggal(rawTanggal)
		tanggalLahir, err := time.Parse("01/02/2006", tanggalNormalized)
		if err != nil {
			fmt.Printf("Gagal parse tanggal untuk %s: %v\n", nama, err)
			continue
		}

		jenisKelamin := row[2].(string)
		alamat := row[3].(string)
		hobi := row[4].(string)

		youthList = append(youthList, Youth{
			Nama:         nama,
			TanggalLahir: tanggalLahir,
			JenisKelamin: jenisKelamin,
			Alamat:       alamat,
			Hobi:         hobi,
		})
	}

	return youthList, nil
}

func normalisasiTanggal(input string) string {
	parsedTime, err := time.Parse("1/2/2006", input)
	if err == nil {
		return parsedTime.Format("01/02/2006")
	}
	return input
}

func cekUlangTahunHariIni(youthList []Youth) []Youth {
	var ulangTahunHariIni []Youth
	hariIni := time.Now()

	for _, p := range youthList {
		if p.TanggalLahir.Month() == hariIni.Month() && p.TanggalLahir.Day() == hariIni.Day() {
			ulangTahunHariIni = append(ulangTahunHariIni, p)
		}
	}

	return ulangTahunHariIni
}

func kirimPesanWhatsApp(to, body, accountSID, authToken, from string) error {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})

	params := &twilioApi.CreateMessageParams{}
	params.SetTo("whatsapp:" + to)
	params.SetFrom("whatsapp:" + from)
	params.SetBody(body)

	resp, err := client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("gagal mengirim pesan: %v", err)
	}

	responseData, _ := json.Marshal(resp)
	log.Printf("Pesan terkirim: %s\n", string(responseData))

	return nil
}

func cekUlangTahunDanKirimNotifikasi() {
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	readRange := "Form Responses 1!B2:F"

	accountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	fromWhatsApp := os.Getenv("TWILIO_FROM_WHATSAPP")
	adminWhatsApp := os.Getenv("TWILIO_TO_WHATSAPP")

	youthList, err := bacaGoogleSheet(spreadsheetID, readRange)
	if err != nil {
		fmt.Printf("Gagal membaca Google Sheets: %v\n", err)
		return
	}

	ulangTahunHariIni := cekUlangTahunHariIni(youthList)

	if len(ulangTahunHariIni) > 0 {
		message := "Hari ini ada yang ulang tahun:\n\n"
		for _, p := range ulangTahunHariIni {
			message += fmt.Sprintf("- %s (Jenis Kelamin: %s, Alamat: %s, Hobi: %s)\n",
				p.Nama, p.JenisKelamin, p.Alamat, p.Hobi)
		}
		err := kirimPesanWhatsApp(adminWhatsApp, message, accountSID, authToken, fromWhatsApp)
		if err != nil {
			fmt.Printf("Gagal mengirim pesan ke admin: %v\n", err)
		}
	} else {
		fmt.Println("Hari ini tidak ada yang berulang tahun.")
	}
}

func main() {
	fmt.Println("Menjalankan uji coba notifikasi...")
	cekUlangTahunDanKirimNotifikasi()

	for {
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())
		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}

		waitDuration := nextRun.Sub(now)
		fmt.Printf("Menunggu hingga waktu berikutnya: %v\n", nextRun)
		time.Sleep(waitDuration)

		cekUlangTahunDanKirimNotifikasi()
	}
}
