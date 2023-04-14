package controllers

import (
	"context"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types/filters"
	_ "github.com/docker/docker/api/types/volume"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/salvatore-081/goup/internal"
	"github.com/salvatore-081/goup/internal/middlewares"
	"github.com/salvatore-081/goup/pkg/models"
)

func Docker(g *gin.RouterGroup, r *internal.Resolver) {
	ListVolumes(g, r)
	InspectVolume(g, r)
}

// @Tags Docker
// @Summary Volume Placeholder
// @Produce  json
// @Success 200 {object} []volume.Volume
// @Failure 500 {object} models.Default
// @Router /docker/volume [get]
// @Security X-API-Key
func ListVolumes(g *gin.RouterGroup, r *internal.Resolver) {
	g.GET("/volume", middlewares.GinAuthMiddleware(r.XAPIKey), func(c *gin.Context) {

		volumes, e := r.DockerClient.VolumeList(context.Background(), filters.Args{})
		if e != nil {
			log.Debug().Err(e).Str("PATH", "docker").Str("SERVICE", "API").Msg("ListVolumes")

			c.JSON(http.StatusInternalServerError, models.Default{
				Message: "unable to retrive volumes",
				Details: e.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, volumes.Volumes)
		return
	})
}

// @Tags Docker
// @Summary Volume Placeholder
// @Produce  json
// @Success 200 {object} volume.Volume
// @Failure 404,500 {object} models.Default
// @Router /docker/volume/{key} [get]
// @Security X-API-Key
// @Param key path string true "volume name"
func InspectVolume(g *gin.RouterGroup, r *internal.Resolver) {
	g.GET("/volume/:key", middlewares.GinAuthMiddleware(r.XAPIKey), func(c *gin.Context) {

		inspect, e := r.DockerClient.VolumeInspect(context.Background(), c.Param("key"))
		if e != nil {
			switch {
			case strings.HasSuffix(e.Error(), "no such volume"):
				c.JSON(http.StatusNotFound, models.Default{
					Message: "no such volume",
				})
				return
			default:
				log.Debug().Err(e).Str("PATH", "docker").Str("SERVICE", "API").Msg("InspectVolume")
				c.JSON(http.StatusInternalServerError, models.Default{
					Message: "unable to retrive volume info",
					Details: e.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusOK, inspect)
		return
	})
}
