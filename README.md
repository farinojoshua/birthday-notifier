# Birthday Notifier

A Golang application to send WhatsApp notifications via Twilio API when there are birthdays from a list stored in Google Sheets.

---

## Features

- Fetches data from Google Sheets.
- Parses and checks for birthdays matching today's date.
- Sends notifications via WhatsApp to a configured admin number using Twilio API.
- Automatically runs daily at 8:00 AM.
