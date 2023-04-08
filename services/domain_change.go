package services

import (
	db "domain0/database"
	"domain0/models"
	mw "domain0/models/web"
	"domain0/modules"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

// @Summary list all domain change requests generated by the user
// @Tags domain
// @Produce json
// @Success 200 {object} mw.Domain{data=[]models.DomainChange}
// @Failure 500 {object} mw.Domain
// @Router /api/v1/domain/change/myapply [get]
func DomainChangeListMyApply(c *fiber.Ctx) error {
	// extract info
	uid := c.Locals("sub").(uint)

	// get domain change list
	var dcList []models.DomainChange
	err := db.DB.Preload("Domain").Preload("User").Where("user_id = ?", uid).Find(&dcList).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(mw.Domain{
			Status: fiber.StatusInternalServerError,
			Errors: "Database error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(mw.Domain{
		Status: fiber.StatusOK,
		Data:   dcList,
	})
}

// @Summary list all domain change requests that the user can approve
// @Tags domain
// @Produce json
// @Success 200 {object} mw.Domain{data=[]models.DomainChange}
// @Failure 500 {object} mw.Domain
// @Router /api/v1/domain/change/myapprove [get]
func DomainChangeListMyApprove(c *fiber.Ctx) error {
	// extract info
	uid := c.Locals("sub").(uint)

	// get domain change list
	var dcList []models.DomainChange
	err := db.DB.Preload("Domain").Preload("User").Where(
		"domain_id IN (SELECT domain_id FROM user_domains WHERE user_id = ? AND role >= ?)",
		uid, models.Owner,
	).Find(&dcList).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(mw.Domain{
			Status: fiber.StatusInternalServerError,
			Errors: "Database error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(mw.Domain{
		Status: fiber.StatusOK,
		Data:   dcList,
	})
}

// @Summary modify domain change request
// @Tags domain
// @Produce json
// @Param id path string true "domain change id"
// @Param opt query string true "operation: accept or reject"
// @Success 200 {object} mw.Domain{data=models.DomainChange}
// @Failure 500 {object} mw.Domain
// @Router /api/v1/domain/change/{id} [put]
func DomainChangeCheck(c *fiber.Ctx) error {
	// extract info
	uid := c.Locals("sub").(uint)

	// get domain change id
	dcId := c.Params("id")
	opt := c.Query("opt")

	// get domain change
	var dc models.DomainChange
	err := db.DB.Where("id = ?", dcId).First(&dc).Error
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(mw.Domain{
			Status: fiber.StatusNotFound,
			Errors: "Domain change not found",
		})
	}

	// check permission
	var ud models.UserDomain
	err = db.DB.Where("domain_id = ? AND user_id = ? AND role >= ?", dc.DomainId, uid, models.Owner).First(&ud).Error
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(mw.Domain{
			Status: fiber.StatusForbidden,
			Errors: "Permission denied",
		})
	}

	// oprate
	if opt == "accept" {
		dc.ActionStatus = models.Approved
		var dcs modules.DnsChangeStruct
		if err := json.Unmarshal([]byte(dc.Operation), &dcs); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(mw.Domain{
				Status: fiber.StatusInternalServerError,
				Errors: "Database error",
			})
		}
		if err := dcs.DnsChangeRestore(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(mw.Domain{
				Status: fiber.StatusInternalServerError,
				Errors: "Database error",
			})
		}
		var err error
		if dc.ActionType == models.Submit {
			err = dcs.Dns.Create()
		} else if dc.ActionType == models.EditDNS {
			err = dcs.Dns.Update()
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(mw.Domain{
				Status: fiber.StatusInternalServerError,
				Errors: "Database error",
			})
		}
	} else if opt == "reject" {
		dc.ActionStatus = models.Rejected
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(mw.Domain{
			Status: fiber.StatusBadRequest,
			Errors: "Invalid opt",
		})
	}

	// update domain change
	err = db.DB.Save(&dc).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(mw.Domain{
			Status: fiber.StatusInternalServerError,
			Errors: "Database error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(mw.Domain{
		Status: fiber.StatusOK,
		Data:   dc,
	})
}
