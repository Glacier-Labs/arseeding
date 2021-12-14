package seeding

import (
	"github.com/everFinance/goar/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) broadcast(c *gin.Context) {
	arid := c.Param("arid")
	txHash, err := utils.Base64Decode(arid)
	if err != nil || len(txHash) != 32 {
		c.JSON(http.StatusBadRequest, "arId incorrect")
		return
	}

	if err := s.jobManager.RegisterJob(arid, jobTypeBroadcast, int64(len(s.peers))); err != nil {
		c.JSON(http.StatusBadGateway, err.Error())
		return
	}

	if err := s.store.PutPendingPool(jobTypeBroadcast, arid); err != nil {
		s.jobManager.UnregisterJob(arid, jobTypeBroadcast)
		c.JSON(http.StatusBadGateway, err.Error())
		return
	}

	c.JSON(http.StatusOK, "ok")
}

func (s *Server) sync(c *gin.Context) {
	arid := c.Param("arid")
	txHash, err := utils.Base64Decode(arid)
	if err != nil || len(txHash) != 32 {
		c.JSON(http.StatusBadRequest, "arId incorrect")
		return
	}

	if err := s.jobManager.RegisterJob(arid, jobTypeSync, int64(len(s.peers))); err != nil {
		c.JSON(http.StatusBadGateway, err.Error())
		return
	}

	if err := s.store.PutPendingPool(jobTypeSync, arid); err != nil {
		s.jobManager.UnregisterJob(arid, jobTypeSync)
		c.JSON(http.StatusBadGateway, err.Error())
		return
	}

	c.JSON(http.StatusOK, "ok")
}

func (s *Server) killJob(c *gin.Context) {
	arid := c.Param("arid")
	jobType := c.Param("jobType")
	if !strings.Contains(jobTypeSync+jobTypeBroadcast, strings.ToLower(jobType)) {
		c.JSON(http.StatusBadRequest, "jobType not exist")
		return
	}
	txHash, err := utils.Base64Decode(arid)
	if err != nil || len(txHash) != 32 {
		c.JSON(http.StatusBadRequest, "arId incorrect")
		return
	}
	err = s.jobManager.CloseJob(arid, jobType)
	if err != nil {
		c.JSON(http.StatusNotFound, err.Error())
	} else {
		c.JSON(http.StatusOK, "ok")
	}
}

func (s *Server) getJob(c *gin.Context) {
	arid := c.Param("arid")
	jobType := c.Param("jobType")
	if !strings.Contains(jobTypeSync+jobTypeBroadcast, strings.ToLower(jobType)) {
		c.JSON(http.StatusBadRequest, "jobType not exist")
		return
	}
	txHash, err := utils.Base64Decode(arid)
	if err != nil || len(txHash) != 32 {
		c.JSON(http.StatusBadRequest, "arId incorrect")
		return
	}
	// get from cache
	job := s.jobManager.GetJob(arid, jobType)
	if job != nil {
		c.JSON(http.StatusOK, job)
		return
	}

	// get from db
	job, err = s.store.LoadJobStatus(jobType, arid)
	if err != nil {
		c.JSON(http.StatusNotFound, err.Error())
	} else {
		c.JSON(http.StatusOK, job)
	}
}

func (s *Server) getCacheJobs(c *gin.Context) {
	c.JSON(http.StatusOK, s.jobManager.GetJobs())
}
