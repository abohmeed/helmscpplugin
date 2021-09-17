This project is one of the labs presented in my course: Helm - The Kubernetes Package Manager Hands-On. 
It is a Helm plugin that is used to automatically package and upload a chart to a remote server over SSH
# The Go version
## As a standalone binary
Build the project by running 
```bash
make build
```
Use the binary as 
```bash
./helmscp -u remote_user -l /path/to/chart -r /path/to/remote/dir -s hostname -p SSH port (22 by default) -k path to the SSH private key (defaults to ~/.ssh/id_rsa)
```