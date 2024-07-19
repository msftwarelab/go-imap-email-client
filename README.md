# go-imap-gmail-client


## Description

This project is a Go-based client for processing Gmail messages using the IMAP protocol. 

## Main Stack and Libraries

- **Programming Language**: Go (Golang)
- **UI Framework**: Fyne
- **Email Client**: github.com/emersion/go-imap/v2, github.com/emersion/go-message/mail
- **JSON Handling**: encoding/json
- **Logging**: log
- **File Handling**: os, bufio

## Features

- **IMAP Integration:** Connect to Gmail and fetch emails.
- **Email Processing:** Process received and sent emails based on specific dates.
- **GUI Application:** User-friendly interface built with Fyne.
- **Date-based Filtering:** Filter and save emails by date.
- **Credentials Management:** Securely save and manage email credentials.
- **Progress Tracking:** Monitor the email fetching and processing status with a visual progress bar.

## Getting Started

1. **Clone the repository:**
```sh
git clone https://github.com/yourusername/go-imap-gmail-client.git
```

2. **Install dependencies:**
```sh
go mod tidy
```

3. **Run the application:**
```sh
go run main.go
```

4. **Setting Up Gmail:**
    
    - This application is currently designed to work with Gmail only.
    - You must use an app password to connect to your Gmail account.
    - Follow these links for more information on setting up an app password:
        - [IMAP Gmail Setup Issue](https://support.google.com/mail/thread/166960012/imap-gmail-com-does-not-recognize-my-password?hl=en)
        - [Sign in with App Passwords](https://support.google.com/accounts/answer/185833#:~:text=to%2520your%2520data.-,sign%2520in%2520with%2520app%2520passwords,-Tip%253A%2520App%2520Passwords)
    - Typically, once you set up 2-Step Verification, you can create an app password directly by visiting [this link](https://myaccount.google.com/apppasswords).

5. **Using the Application:**
    
    - Save your credentials using your Gmail address and app password.
    - Once saved, you can input the desired date in the GUI to generate the text file.
    - You can add several accounts, and all email histories will be saved in one text file in the order they were added.
    - Ensure the text file is generated correctly.

## Note for Windows Users

The first compilation of Fyne on Windows can take up to 10 minutes, depending on your hardware. Subsequent builds will be much faster.

## 🤝 Contributing

Feel free to submit issues, fork the repository, and send pull requests. Contributions to enhance the project are welcome.


## 🔮 Future Plans

- [ ]  Explore direct Gmail login using just the Gmail address and password
- [ ]  Improve GUI for better user experience (feedback and suggestions are welcome!)
- [ ]  Expand support to include other mail servers beyond Gmail
- [ ]  Add detailed documentation and tutorials
