package main

import (
	"archive/zip"
	"compress/flate"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {

	LOG_LEVEL := flag.String("LOG_LEVEL", "MISSING", "log level")
	TIME := flag.String("TIME", "01:00", "UTC time in which to perform the backup in the format hh:mm")
	MAX_RETENTION := flag.Uint("MAX_RETENTION", 14, "backup(s) older then MAX_RETENTION will be deleted")
	BACKUP_SIZE_WARNING := flag.Uint("BACKUP_SIZE_WARNING", 100, "if a backup size is greater then this value, in mb, a warning level log will be printed")
	flag.Parse()

	logOutput := zerolog.ConsoleWriter{Out: os.Stdout}
	logOutput.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("|%s|", i))
	}

	logOutput.FormatTimestamp = func(i interface{}) string {
		return ""
	}

	log.Logger = zerolog.New(logOutput)

	log.Info().Msg("goup starting up")

	// LOGGER
	var e error
	var l zerolog.Level

	if *LOG_LEVEL == "MISSING" {
		log.Info().Msg("missing LOG_LEVEL, defaulting to DEBUG")
		l = 0
	} else {
		l, e = zerolog.ParseLevel(strings.ToLower(*LOG_LEVEL))
		if e != nil {
			log.Info().Err(e).Msg(fmt.Sprintf("unknown LOG_LEVEL: %s, defaulting to DEBUG", *LOG_LEVEL))
			l = 0
		}
	}
	zerolog.SetGlobalLevel(l)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	configTimeSplit := strings.Split(*TIME, ":")
	h, e := strconv.Atoi(configTimeSplit[0])
	if e != nil {
		log.Fatal().Err(e).Msg("")
	}
	m, e := strconv.Atoi(configTimeSplit[1])
	if e != nil {
		log.Fatal().Err(e).Msg("")
	}
	if h < 0 || h > 24 || m < 0 || m > 60 {
		log.Fatal().Err(errors.New("invalid time")).Msg("")
	}

	now := time.Now()
	scheduled_time := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, time.UTC)

	if scheduled_time.Before(time.Now()) {
		scheduled_time = scheduled_time.Add(24 * time.Hour)
	}

	log.Info().Msg("goup backup scheduled to run every day at " + scheduled_time.Format(time.RFC3339)[11:])

	schedule_timer := time.NewTimer(time.Until(scheduled_time))

	wg := sync.WaitGroup{}

	for {
		select {
		case <-schedule_timer.C:
			wg.Add(1)
			go func(current_schedule string) {
				defer wg.Done()

				log.Info().Msg("[BACKUP] starting backup")

				volumes, e := ioutil.ReadDir("./volumes")
				if e != nil {
					log.Fatal().Err(e).Msg("")
				}

				e = os.Mkdir("./data/"+current_schedule, os.ModePerm)
				if e != nil {
					log.Fatal().Err(e).Msg("")
				}

				for _, volume := range volumes {
					volumesWg := sync.WaitGroup{}
					volumesWg.Add(1)

					go func(volume fs.FileInfo) {
						defer volumesWg.Done()

						fileName := "./data/" + current_schedule + "/" + volume.Name() + ".zip"

						z, e := os.Create(fileName)
						if e != nil {
							log.Error().Err(e).Msg("")
							return
						}
						defer z.Close()

						zw := zip.NewWriter(z)
						zw.RegisterCompressor(8, func(out io.Writer) (io.WriteCloser, error) {
							return flate.NewWriter(out, flate.BestCompression)
						})
						defer zw.Close()
						defer printSize(fileName, int32(*BACKUP_SIZE_WARNING))

						e = filepath.WalkDir("volumes/"+volume.Name(), func(path string, d fs.DirEntry, e error) error {
							if e != nil {
								return e
							}

							if d.IsDir() {
								return nil
							}

							file, e := os.Open(path)
							if e != nil {
								return e
							}
							defer file.Close()

							f, e := zw.Create(path[(len("volumes/") + len(volume.Name())):])
							if e != nil {
								return e
							}

							_, e = io.Copy(f, file)

							return e
						})
						if e != nil {
							log.Error().Err(e).Msg("")
						}
					}(volume)

					volumesWg.Wait()
				}

				if len(volumes) < 1 {
					log.Warn().Msg("[BACKUP] no volume found")
				} else {
					log.Info().Msg("[BACKUP] backup completed")
				}
			}(scheduled_time.Format(time.RFC3339))

			wg.Add(1)
			go func(current_schedule string) {
				defer wg.Done()
				log.Info().Msg(fmt.Sprintf("[DELETE] removing any backup older then %d days", *MAX_RETENTION))

				data, e := ioutil.ReadDir("./data")
				if e != nil {
					log.Error().Err(e).Msg("")
					return
				}

				threshold := time.Now().Add(-(time.Duration(*MAX_RETENTION) * 24 * time.Hour)).Format(time.RFC3339)

				removed := 0
				errors := 0

				for _, d := range data[:] {
					if d.Name() < threshold {
						e := os.RemoveAll("./data/" + d.Name())
						if e != nil {
							errors++
							log.Error().Err(e).Msg(d.Name())
						} else {
							removed++
							log.Info().Msg("[DELETE] " + d.Name() + " removed")
						}
					}
				}

				if removed == 0 && errors == 0 {
					log.Info().Msg("[DELETE] there is no backup that needs to be removed")
					return
				}

				log.Info().Msg(fmt.Sprintf("[DELETE] removed %d backup(s) with %d error(s)", removed, errors))

			}(scheduled_time.Format(time.RFC3339))

			scheduled_time = scheduled_time.Add(24 * time.Hour)
			schedule_timer.Reset(time.Until(scheduled_time))

		case <-stop:
			schedule_timer.Stop()
			log.Info().Msg("waiting for current the process to complete")
			wg.Wait()
			log.Info().Msg("exit")
			return
		}
	}
}

func printSize(f string, max int32) {
	fi, e := os.Stat(f)
	if e != nil {
		return
	}

	mb_size := math.Pow(float64(fi.Size()), float64(1)/6)

	label := fmt.Sprintf("[BACKUP] %s - %0.fmb", f[28:], mb_size)

	if mb_size > float64(max) {
		log.Warn().Msg(label)
	} else {

		log.Info().Msg(label)
	}
}
