package configmanagement

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"unified-ai-eval-platform/backend/internal/datastore" // Adjust import path as necessary

	"github.com/gin-gonic/gin"
)

// CreateVendorConfigHandler handles the creation of a new vendor configuration.
func CreateVendorConfigHandler(c *gin.Context) {
	var vc datastore.VendorConfig
	if err := c.ShouldBindJSON(&vc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Basic validation
	if vc.Name == "" || vc.APIType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name and API Type are required fields"})
		return
	}

	// Ensure JSON fields are valid if provided, or default to null/empty JSON object
	if vc.SupportedModels != nil && len(vc.SupportedModels) > 0 {
		if !json.Valid(vc.SupportedModels) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "supported_models is not valid JSON"})
			return
		}
	} else {
		vc.SupportedModels = json.RawMessage("null")
	}

	if vc.OtherConfigs != nil && len(vc.OtherConfigs) > 0 {
		if !json.Valid(vc.OtherConfigs) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "other_configs is not valid JSON"})
			return
		}
	} else {
		vc.OtherConfigs = json.RawMessage("null")
	}


	id, err := datastore.CreateVendorConfig(&vc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create vendor config: " + err.Error()})
		return
	}

	vc.ID = id // Set the ID in the response object
	c.JSON(http.StatusCreated, vc)
}

// GetVendorConfigHandler retrieves a specific vendor configuration by its ID.
func GetVendorConfigHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor config ID format"})
		return
	}

	vc, err := datastore.GetVendorConfig(id)
	if err != nil {
		if err.Error().Contains("not found") { // More robust error checking is preferred
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve vendor config: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, vc)
}

// UpdateVendorConfigHandler updates an existing vendor configuration.
func UpdateVendorConfigHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor config ID format"})
		return
	}

	var vc datastore.VendorConfig
	if err := c.ShouldBindJSON(&vc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	vc.ID = id // Ensure the ID from the path is used

	// Basic validation
	if vc.Name == "" || vc.APIType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name and API Type are required fields"})
		return
	}
	
	// Ensure JSON fields are valid if provided, or default to null/empty JSON object
	if vc.SupportedModels != nil && len(vc.SupportedModels) > 0 {
		if !json.Valid(vc.SupportedModels) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "supported_models is not valid JSON"})
			return
		}
	} else {
		// If the client sends an empty or null value for an optional JSON field,
		// and you want to preserve existing data in the DB if not provided,
		// you might need to fetch the existing record first and merge.
		// For simplicity, here we'll just set it to null if not provided or invalid.
		vc.SupportedModels = json.RawMessage("null")
	}

	if vc.OtherConfigs != nil && len(vc.OtherConfigs) > 0 {
		if !json.Valid(vc.OtherConfigs) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "other_configs is not valid JSON"})
			return
		}
	} else {
		vc.OtherConfigs = json.RawMessage("null")
	}


	err = datastore.UpdateVendorConfig(&vc)
	if err != nil {
		if err.Error().Contains("not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vendor config: " + err.Error()})
		}
		return
	}

	// Fetch the updated record to return it, as UpdateVendorConfig doesn't return the object
	updatedVc, err := datastore.GetVendorConfig(id)
	if err != nil {
		// This case should ideally not happen if update was successful
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated vendor config: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedVc)
}

// DeleteVendorConfigHandler deletes a vendor configuration by its ID.
func DeleteVendorConfigHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vendor config ID format"})
		return
	}

	err = datastore.DeleteVendorConfig(id)
	if err != nil {
		if err.Error().Contains("not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete vendor config: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vendor config deleted successfully"})
}

// ListVendorConfigsHandler lists vendor configurations, optionally filtered by api_type.
func ListVendorConfigsHandler(c *gin.Context) {
	apiType := c.Query("api_type") // e.g., /vendors?api_type=ASR

	vcs, err := datastore.ListVendorConfigs(apiType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list vendor configs: " + err.Error()})
		return
	}

	if vcs == nil {
		vcs = []*datastore.VendorConfig{} // Return empty array instead of null
	}

	c.JSON(http.StatusOK, vcs)
}

// Note: The datastore.DB connection needs to be initialized in main.go
// and potentially passed to these handlers if not using a global variable (which is not recommended for production).
// For this subtask, we assume datastore.DB is accessible as per vendor_store.go's current structure.
// Example of how DB might be passed if handlers were methods on a struct:
/*
type VendorHandler struct {
    DB *sql.DB
}

func (vh *VendorHandler) CreateVendorConfigHandler(c *gin.Context) {
    // use vh.DB
}
*/
// For now, the global datastore.DB is implicitly used.
func InitHandlers(db *sql.DB) {
	// This function could be used to pass the DB connection to the handlers
	// For now, it's not strictly necessary as the datastore package uses a global DB variable.
	// However, it's good practice to have an explicit initialization step.
	datastore.DB = db
}
