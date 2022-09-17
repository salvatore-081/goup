# goup

## What is it

Free and open-source backup tool for docker volumes written in Go.

## Quick start

Run in a terminal:

`docker run -v vol1:/app/volumes/vol1 -v vol2:/app/volumes/vol2 -v vol3:/app/volumes/vol3 -v ./data:/app/data/ --name goup -e LOG_LEVEL=INFO -e TIME=01:00 -e MAX_RETENTION=14 -e BACKUP_SIZE_WARNING=100 salvatoreemilio/goup:latest`

Every docker volume that you wish to backup has to be mounted in the /app/volumes folder.

Or use a docker-compose.

## Configuration

- **TIME** set the time, in UTC, in which the backup will run
- Every backup older then **MAX_RETENTION** will get deleted automatically durin the backup run
- You can set up a warning log with **BACKUP_SIZE_WARNING** to be alerted if a backup is greater, in MB, then your setting

## Examples

- [compose.yaml](./examples/compose.yaml)

## License

[Apache License 2.0](./LICENSE)
