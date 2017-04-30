package main

import (
	"flag"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/galexrt/srcds_exporter/models"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsAddr = flag.String("metrics.listen-address", ":9137", "The address to listen on for HTTP requests.")
)

var (
	serverIdentification string
)

// Metric vars
var (
	metricServerMap           prometheus.Counter
	metricPlayerCountCurrent  prometheus.Gauge
	metricPlayerCountMax      prometheus.Gauge
	metricPlayersMetrics      = make(map[int]prometheus.Counter)
	playersMetricsToBeRemoved = make(map[int]prometheus.Counter)
)

func initMetrics(status models.Status) {
	metricServerMap = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "gameserver",
		Subsystem: "map",
		Name:      "current",
		Help:      "Current map played.",
		ConstLabels: map[string]string{
			"server": serverIdentification,
			"map":    status.Map,
		},
	})
	metricPlayerCountCurrent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "gameserver",
		Subsystem: "player_count",
		Name:      "current",
		Help:      "Current player count on the server.",
		ConstLabels: map[string]string{
			"server": serverIdentification,
		},
	})
	metricPlayerCountMax = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "gameserver",
		Subsystem: "player_count",
		Name:      "max",
		Help:      "Maximum player count on the server.",
		ConstLabels: map[string]string{
			"server": serverIdentification,
		},
	})
	prometheus.MustRegister(metricServerMap)
	prometheus.MustRegister(metricPlayerCountCurrent)
	prometheus.MustRegister(metricPlayerCountMax)
	updatePlayersMetrics(status.Players)
	go func() {
		for {
			<-time.After(3 * time.Minute)
			cleanupPlayersMetrics()
		}
	}()
}

func updateMetrics(status models.Status) {
	if !strings.Contains(metricServerMap.Desc().String(), "map=\""+status.Map+"\"") {
		log.WithFields(logrus.Fields{
			"map": status.Map,
		}).Debug("exporter: map name update required")
		prometheus.Unregister(metricServerMap)
		metricServerMap = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "gameserver",
			Subsystem: "map",
			Name:      "current",
			Help:      "Current map played.",
			ConstLabels: map[string]string{
				"server": serverIdentification,
				"map":    status.Map,
			},
		})
		metricServerMap.Inc()
		prometheus.MustRegister(metricServerMap)
	} else {
		log.WithFields(logrus.Fields{
			"map": status.Map,
		}).Debug("exporter: no map name update required")
	}
	metricPlayerCountCurrent.Set(float64(status.PlayerCount.Current))
	metricPlayerCountMax.Set(float64(status.PlayerCount.Max))
	updatePlayersMetrics(status.Players)
}

func updatePlayersMetrics(players map[int]models.Player) {
	log.Debugln("updatePlayersMetrics: called")
	for userID, player := range players {
		if _, ok := metricPlayersMetrics[userID]; !ok {
			metricPlayersMetrics[userID] = prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: "gameserver",
				Subsystem: "players",
				Name:      "current",
				Help:      "Current users by Steam ID playing on the server.",
				ConstLabels: map[string]string{
					"server":  serverIdentification,
					"steamid": player.SteamID,
				},
			})
			metricPlayersMetrics[userID].Inc()
			prometheus.MustRegister(metricPlayersMetrics[userID])
			log.WithFields(logrus.Fields{
				"userid": userID,
				"player": player,
			}).Debug("updatePlayersMetrics: added user metric")
		} else {
			log.WithFields(logrus.Fields{
				"userid": userID,
				"player": player,
			}).Debug("updatePlayersMetrics: user already has metric")
		}
	}
	for userID, metric := range metricPlayersMetrics {
		if _, ok := players[userID]; !ok {
			metric.Desc()
			playersMetricsToBeRemoved[userID] = metric
			delete(metricPlayersMetrics, userID)
			log.WithFields(logrus.Fields{
				"userid": userID,
			}).Debug("updatePlayersMetrics: removed user metric")
		}
	}
}

func cleanupPlayersMetrics() {
	for _, metric := range playersMetricsToBeRemoved {
		prometheus.Unregister(metric)
	}
}
