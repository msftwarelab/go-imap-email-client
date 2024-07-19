package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const (
	appName    = "gmail_box_api"
	appVersion = "v1.0"
	credsFile  = "credentials.json"
)

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Email Processor")
	myWindow.Resize(fyne.NewSize(800, 600))

	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("Email")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")

	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("Date (YYYY-MM-DD)")	

	progressBar := widget.NewProgressBar()
	progressLabel := widget.NewLabel("")

	credsList, err := loadCredentials()
	if err != nil {
		credsList = []Credentials{} // Initialize an empty list if the file doesn't exist
	}

	emailsContainer := container.NewVBox()
	updateEmailList(emailsContainer, credsList)

	form := container.NewVBox(
		emailEntry,
		passwordEntry,
		dateEntry, // Add the date entry here
		widget.NewButton("Save Credentials", func() {
			if emailEntry.Text == "" {
				dialog.ShowInformation("Error", "Email cannot be empty", myWindow)
				return
			}
			if passwordEntry.Text == "" {
				dialog.ShowInformation("Error", "Password cannot be empty", myWindow)
				return
			}
			creds := Credentials{
				Email:    emailEntry.Text,
				Password: passwordEntry.Text,
			}
			err := saveCredentials(creds)
			if err != nil {
				dialog.ShowInformation("Error", err.Error(), myWindow)
				return
			}
			// Update credsList directly
			credsList = append(credsList, creds)
			updateEmailList(emailsContainer, credsList)
			dialog.ShowInformation("Success", "Credentials saved successfully!", myWindow)
		}),
		widget.NewButton("Generate Text File", func() {
			if len(credsList) == 0 {
				dialog.ShowInformation("Error", "No credentials found", myWindow)
				return
			}
		
			specificDate := dateEntry.Text
			if specificDate == "" {
				dialog.ShowInformation("Error", "Please enter a date", myWindow)
				return
			}
		
			go func() {
				progressBar.SetValue(0)
				totalCredentials := len(credsList)
				progressStep := 1.0 / float64(totalCredentials)
		
				for i, creds := range credsList {
					progressLabel.SetText(fmt.Sprintf("Fetching received emails for %s...", creds.Email))
					err := processEmailsHandler(creds.Email, creds.Password, specificDate, progressBar, progressLabel, myWindow)
					if err != nil {
						dialog.ShowInformation("Error", err.Error(), myWindow)
						return
					}
					progressBar.SetValue(progressStep * float64(i+1))
				}
				progressLabel.SetText("Done")
				dialog.ShowInformation("Success", "Text file generated successfully!", myWindow)
				progressBar.SetValue(1)
			}()
		}),
		progressLabel,
		progressBar,
		emailsContainer,
	)

	myWindow.SetContent(form)
	myWindow.ShowAndRun()
}

// saveCredentials saves the provided credentials to the credentials file.
// It first loads existing credentials, appends the new one, and then writes them all back to the file.
func saveCredentials(creds Credentials) error {
	var credsList []Credentials

	// Load existing credentials if they exist
	file, err := os.Open(credsFile)
	if err == nil {
		defer file.Close()
		err = json.NewDecoder(file).Decode(&credsList)
		if err != nil {
			return err
		}
	}

	// Append new credentials
	credsList = append(credsList, creds)

	// Save credentials to file
	file, err = os.Create(credsFile)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(credsList)
}

// updateEmailList updates the email list container with the provided credentials list.
func updateEmailList(container *fyne.Container, credsList []Credentials) {
	container.Objects = nil
	for _, creds := range credsList {
		container.Add(widget.NewLabel(creds.Email))
	}
	container.Refresh()
}

// loadCredentials loads the credentials from the credentials file.
// It returns an empty list if the file does not exist.
func loadCredentials() ([]Credentials, error) {
	var credsList []Credentials
	file, err := os.Open(credsFile)
	if os.IsNotExist(err) {
		return credsList, nil // Return an empty list if the file doesn't exist
	} else if err != nil {
		return nil, err
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&credsList)
	return credsList, err
}

// processEmailsHandler connects to the IMAP server, logs in, and processes emails from the specified date.
// It handles both received and sent emails, updating the progress bar as it processes.
func processEmailsHandler(email, password, specificDate string, progressBar *widget.ProgressBar, progressLabel *widget.Label, myWindow fyne.Window) error {
	imapserver := "imap.gmail.com:993"
	log.Println("Processing emails for", email)

	log.Println("Connecting to server...")

	c, err := imapclient.DialTLS(imapserver, nil)
	if err != nil {
		return err
	}
	defer c.Close()

	// Login
	if err := c.Login(email, password).Wait(); err != nil {
		return fmt.Errorf("failed to login: %v", err)
	}
	log.Println("Logged in")

	log.Println("Starting to process emails...")
	var messageEntries []MessageEntry
	progressLabel.SetText(fmt.Sprintf("Fetching received emails for %s...", email))
	receivedMessageEntries := processEmails(c, "INBOX", specificDate, email, progressBar, myWindow)
	messageEntries = append(messageEntries, receivedMessageEntries...)

	progressLabel.SetText(fmt.Sprintf("Fetching sent emails for %s...", email))
	sentMessageEntries := processEmails(c, "[Gmail]/Sent Mail", specificDate, email, progressBar, myWindow)
	messageEntries = append(messageEntries, sentMessageEntries...)

	err = writeToFile(messageEntries, specificDate)
	if err != nil {
		return err
	}
	log.Println("Emails processed and written to file.")

	return nil
}

// processEmails fetches and processes emails from the specified mailbox for a specific date.
// It filters out irrelevant emails and updates the progress bar.
func processEmails(c *imapclient.Client, mailboxName string, specificDate string, useremail string, progressBar *widget.ProgressBar, myWindow fyne.Window) []MessageEntry {
	var messageEntries []MessageEntry
	log.Println("Fetching " + mailboxName + " messages...")

	mbox, err := c.Select(mailboxName, nil).Wait()
	if err != nil {
		log.Fatalf("failed to select mailbox: %v", err)
	}
	log.Printf("Mailbox %s selected.", mailboxName)

	seqSet := imap.SeqSetRange(mbox.NumMessages - 500, mbox.NumMessages)
	fetchOptions := &imap.FetchOptions{
		UID:         true,
		Flags:       true,
		Envelope:    true,
		BodySection: []*imap.FetchItemBodySection{{}},
	}

	messages := c.Fetch(seqSet, fetchOptions)

	log.Println("Messages fetched, starting to process each message...")

	messageCount := 501

	progressStep := 1.0 / float64(messageCount)
	currentProgress := 0.0

	filterKeywords := []string{"voice.google.com", "noreply", "donotreply", "no-reply", "mailtrack.io", "dice.com", "medium.com", "jobot.com", "lensa.com", "experteer.com", 
														"calendly.com",	"linkedin.com", "indeed.com", "glassdoor.com"}

	for {
		msg := messages.Next()
		if msg == nil {
			break
		}

		content, date, from, fromName, to, toName := "", time.Now(), "", "", "", ""
		var attachments []AttachmentInfo
		mailType := ""

		validMessage := false

		for {
			item := msg.Next()
			if item == nil {
				break
			}

			switch item := item.(type) {
			case imapclient.FetchItemDataEnvelope:
				date = item.Envelope.Date

				// Filter by specific date
				if date.Format("2006-01-02") != specificDate {
					break
				}

				validMessage = true

				if len(item.Envelope.From) > 0 {
					from = item.Envelope.From[0].Mailbox + "@" + item.Envelope.From[0].Host
					fromName = item.Envelope.From[0].Name

					for _, keyword := range filterKeywords {
						if strings.Contains(from, keyword) {
							validMessage = false
							break
						}
					}
				}
				if len(item.Envelope.To) > 0 {
					to = item.Envelope.To[0].Mailbox + "@" + item.Envelope.To[0].Host
					toName = item.Envelope.To[0].Name
				}

				if strings.Contains(from, useremail) {
					mailType = "Sent"
				} else {
					mailType = "Received"
				}

			case imapclient.FetchItemDataBodySection:
				if !validMessage {
					break
				}
				mr, err := mail.CreateReader(item.Literal)
				if err != nil {
					log.Fatal(err)
				}
				for {
					p, err := mr.NextPart()
					if err == io.EOF {
						break
					} else if err != nil {
						log.Fatal(err)
					}

					switch h := p.Header.(type) {
					case *mail.InlineHeader:
						contentType, _, err := h.ContentType()
						b, _ := io.ReadAll(p.Body)
						if err != nil {
							log.Fatal(err)
						}

						if contentType == "text/plain" {
							content = string(b)
						} else if contentType == "text/html" {
							if content == "" {
								content = extractContent(string(b))
							}
						}
					case *mail.AttachmentHeader:
						filename, _ := h.Filename()
						attachments = append(attachments, AttachmentInfo{
							Filename: filename,
						})
					}
				}
			}
		}

		if validMessage {
			entry := MessageEntry{
				Date:        date,
				From:        from,
				FromName:    fromName,
				To:          to,
				ToName:      toName,
				Content:     content,
				Attachments: attachments,
				MailType:    mailType,
			}
			messageEntries = append(messageEntries, entry)
		}

		// Update progress bar and show fetching message
		currentProgress += progressStep
		progressBar.SetValue(currentProgress)
	}

	log.Println("Finished processing messages.")
	return messageEntries
}

// extractContent removes HTML tags, URLs, and extra spaces from the input string.
func extractContent(input string) string {
	htmlPattern := `(?s)<html[^>]*>.*?</html>`
	htmlReg := regexp.MustCompile(htmlPattern)
	htmlContent := htmlReg.FindString(input)

	bodyPattern := `(?s)<body[^>]*>.*?</body>`
	bodyReg := regexp.MustCompile(bodyPattern)
	bodyContent := bodyReg.FindString(htmlContent)

	textContent := stripTags(bodyContent)

	// Remove URLs
	re := regexp.MustCompile(`https?://[^\s]+`)
	textContent = re.ReplaceAllString(textContent, "")

	// Remove extra spaces (more than two consecutive spaces) and newlines
	textContent = strings.ReplaceAll(textContent, "\n", " ")
	re = regexp.MustCompile(`\s{2,}`)
	textContent = re.ReplaceAllString(textContent, " ")

	return strings.TrimSpace(textContent)
}

// stripTags removes all HTML tags from the input string.
func stripTags(input string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(input, "")
}

// writeToFile writes the provided message entries to a text file, sorted by date.
// If the file already exists, it appends the entries to the file.
func writeToFile(entries []MessageEntry, specificDate string) error {
	// Sort the entries by date in ascending order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.Before(entries[j].Date)
	})

	// Write the data to file
	filename := fmt.Sprintf("%s.txt", strings.ReplaceAll(specificDate, "-", ""))
	var file *os.File

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		file, err = os.Create(filename)
		if err != nil {
			return err
		}
		log.Printf("Creating new file %s...", filename)
	} else {
		file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		log.Printf("Appending to existing file %s...", filename)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, entry := range entries {
		writer.WriteString(fmt.Sprintf("========================================================\n\n"))
		writer.WriteString(fmt.Sprintf("Mail Type: %s\n", entry.MailType))
		writer.WriteString(fmt.Sprintf("Sender address: %s\n", entry.From))
		writer.WriteString(fmt.Sprintf("Sender name: %s\n", entry.FromName))
		writer.WriteString(fmt.Sprintf("Receiver address: %s\n", entry.To))
		writer.WriteString(fmt.Sprintf("Receiver name: %s\n", entry.ToName))
		writer.WriteString(fmt.Sprintf("Date: %s\n", entry.Date.Format("2006-01-02 15:04:05")))
		for i, attachment := range entry.Attachments {
			writer.WriteString(fmt.Sprintf("Attached file%d name: %s\n", i+1, attachment.Filename))
		}
		writer.WriteString(fmt.Sprintf("Content: %s\n\n", entry.Content))
	}
	writer.Flush()
	log.Printf("Finished writing entries to file %s.", filename)
	return nil
}

type MessageEntry struct {
	From        string
	FromName    string
	To          string
	ToName      string
	Date        time.Time
	Content     string
	Attachments []AttachmentInfo
	MailType    string
}

type AttachmentInfo struct {
	Filename string
	Size     int64
}
