package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/salvatore-081/goup/internal"
	"github.com/salvatore-081/goup/internal/middlewares"
	"github.com/salvatore-081/goup/pkg/models"
)

func GoUp(g *gin.RouterGroup, r *internal.Resolver) {
	AddVolumeToBackupList(g, r)
	RemoveVolumeFromBackupList(g, r)
	GetBackupList(g, r)
}

// @Tags GoUp
// @Summary GoUp Placeholder
// @Produce  json
// @Success 201 {object} models.AddVolumeToBackupBody
// @Failure 400,404,500 {object} models.Default
// @Router /goup [post] models.AddVolumeToBackupBody
// @Param message body models.AddVolumeToBackupBody true "volume"
// @Security X-API-Key
func AddVolumeToBackupList(g *gin.RouterGroup, r *internal.Resolver) {
	g.POST("/", middlewares.GinAuthMiddleware(r.XAPIKey), func(c *gin.Context) {
		var body models.AddVolumeToBackupBody
		if e := c.ShouldBindJSON(&body); e != nil {
			c.JSON(http.StatusBadRequest,
				models.Default{
					Message: e.Error(),
				})
			return
		}

		_, e := r.DockerClient.VolumeInspect(context.Background(), body.Name)
		if e != nil {
			switch {
			case strings.HasSuffix(e.Error(), "no such volume"):
				c.JSON(http.StatusNotFound, models.Default{
					Message: "no such volume",
				})
				return
			default:
				log.Debug().Err(e).Str("PATH", "goup").Str("SERVICE", "API").Msg("AddVolumeToBackupList.VolumeInspect")
				c.JSON(http.StatusInternalServerError, models.Default{
					Message: "unable to retrive volume info",
					Details: e.Error(),
				})
				return
			}
		}

		e = r.BadgerDB.Update(func(txn *badger.Txn) error {

			item, e := txn.Get([]byte("volumes"))
			if e != nil {
				return e
			}

			v, e := item.ValueCopy(nil)
			if e != nil {
				return e
			}

			var volumes []string

			e = json.Unmarshal(v, &volumes)
			if e != nil {
				return e
			}

			for _, volume := range volumes {
				if volume == body.Name {
					return &models.HTTPError{Status: http.StatusConflict}
				}
			}

			volumes = append(volumes, body.Name)

			b, e := json.Marshal(volumes)
			if e != nil {
				return e
			}

			e = txn.SetEntry(badger.NewEntry([]byte("volumes"), b))
			if e != nil {
				return e
			}
			return nil
		})
		if e != nil {
			log.Debug().Err(e).Str("PATH", "goup").Str("SERVICE", "API").Msg("AddVolumeToBackupList.Update")
			switch {
			case models.HTTPError{Status: http.StatusConflict}.Equal(e):
				c.JSON(http.StatusConflict, models.Default{
					Message: "the volume is already configured for backup",
				})
				return
			default:
				c.JSON(http.StatusInternalServerError, models.Default{
					Message: "unexpected error",
					Details: e.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusCreated, body)
		return
	})
}

// @Tags GoUp
// @Summary GoUp Placeholder
// @Produce  plain/text
// @Success 200 {string} string {key}
// @Failure 400,404,500 {object} models.Default
// @Router /goup/{key} [delete]
// @Param key path string true "volume name"
// @Security X-API-Key
func RemoveVolumeFromBackupList(g *gin.RouterGroup, r *internal.Resolver) {
	g.DELETE("/:key", middlewares.GinAuthMiddleware(r.XAPIKey), func(c *gin.Context) {
		if len(c.Param("key")) < 1 {
			c.JSON(http.StatusBadRequest,
				models.Default{
					Message: "path parameter volume name missing",
				})
			return
		}

		_, e := r.DockerClient.VolumeInspect(context.Background(), c.Param("key"))
		if e != nil {
			switch {
			case strings.HasSuffix(e.Error(), "no such volume"):
				c.JSON(http.StatusNotFound, models.Default{
					Message: "no such volume",
				})
				return
			default:
				log.Debug().Err(e).Str("PATH", "goup").Str("SERVICE", "API").Msg("RemoveVolumeFromBackupList.VolumeInspect")
				c.JSON(http.StatusInternalServerError, models.Default{
					Message: "unable to retrive volume info",
					Details: e.Error(),
				})
				return
			}
		}

		e = r.BadgerDB.Update(func(txn *badger.Txn) error {

			item, e := txn.Get([]byte("volumes"))
			if e != nil {
				return e
			}

			v, e := item.ValueCopy(nil)
			if e != nil {
				return e
			}

			var volumes []string

			e = json.Unmarshal(v, &volumes)
			if e != nil {
				return e
			}

			indexToRemove := -1
			for i, volume := range volumes {
				if volume == c.Param("key") {
					indexToRemove = i
				}
			}

			if indexToRemove < 0 {
				return &models.HTTPError{Status: http.StatusNotFound}
			}

			updatedVolumes := make([]string, 0)
			updatedVolumes = append(updatedVolumes, volumes[:indexToRemove]...)
			updatedVolumes = append(updatedVolumes, volumes[indexToRemove+1:]...)

			b, e := json.Marshal(updatedVolumes)
			if e != nil {
				return e
			}

			e = txn.SetEntry(badger.NewEntry([]byte("volumes"), b))
			if e != nil {
				return e
			}
			return nil
		})
		if e != nil {
			switch {
			case models.HTTPError{Status: http.StatusNotFound}.Equal(e):
				c.JSON(http.StatusNotFound, models.Default{
					Message: "volume is not set to backup",
				})
				return
			default:
				log.Debug().Err(e).Str("PATH", "goup").Str("SERVICE", "API").Msg("RemoveVolumeFromBackupList.Update")
				c.JSON(http.StatusInternalServerError, models.Default{
					Message: "unexpected error",
					Details: e.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusOK, c.Param("key"))
		return
	})
}

// @Tags GoUp
// @Summary GoUp Placeholder
// @Produce  json
// @Success 200 {object} []string
// @Failure 500 {object} models.Default
// @Router /goup [get]
// @Security X-API-Key
func GetBackupList(g *gin.RouterGroup, r *internal.Resolver) {

	g.GET("/", middlewares.GinAuthMiddleware(r.XAPIKey), func(c *gin.Context) {
		var volumes []string

		e := r.BadgerDB.View(func(txn *badger.Txn) error {
			item, e := txn.Get([]byte("volumes"))
			if e != nil {
				return e
			}

			v, e := item.ValueCopy(nil)
			if e != nil {
				return e
			}

			e = json.Unmarshal(v, &volumes)
			if e != nil {
				return e
			}

			return nil
		})
		if e != nil {
			switch {
			default:
				log.Debug().Err(e).Str("PATH", "goup").Str("SERVICE", "API").Msg("GetBackupList.View")
				c.JSON(http.StatusInternalServerError, models.Default{
					Message: "unexpected error",
					Details: e.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusOK, volumes)
		return
	})
}
