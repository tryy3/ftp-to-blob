# ftp-to-blob
This project started when I wanted automatic backups to azure.

We were only able to do automatic backup through FTP, which usually isn't a problem.
But the problem is that Azure doesn't have support for FTP.

The solution was to make a FTP server that has a backend of Azure blob storage.

Luckily most of the work was done with https://github.com/goftp/server that makes it really easy to make a FTP server and allows you to implement a custom File Driver. So all I had to do was to use azure blob storage SDK and then implement a File Driver for it.

## Disclaimer
This is a project that was kinda rushed and have a specific purpose, to make it possible to have a FTP server that when we save files on it, it saves them to azure blob storage.

There is probably plenty of bugs, especially when reading files. Most of the work has gone towards making sure that saving files works.

I will happily fix any bugs that is found but I wouldn't recommend using this project in production.