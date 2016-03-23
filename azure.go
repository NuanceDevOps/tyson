package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	storageAPI "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
)

// Credentials represents a set of Azure credentials for a service principal
// All fields are required
type Credentials struct {
	SubscriptionID string
	ClientID       string
	ClientSecret   string
	TenantID       string
}

// NewCredentialsFromFile will create a new instance of an azure.Credentials object from a JSON formatted file
func NewCredentialsFromFile(path string) Credentials {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	var creds Credentials
	json.Unmarshal(file, &creds)
	return creds
}

// AzureClient represents multiple Azure Resource Manager providers.
type AzureClient struct {
	storageAccounts storage.AccountsClient
	virtualMachines compute.VirtualMachinesClient
}

// NewAzureClient is a helper function for creating an Azure compute client to ARM.
func NewAzureClient(creds Credentials) (AzureClient, error) {

	var c AzureClient
	oauthConfig, err := azure.PublicCloud.OAuthConfigForTenant(creds.TenantID)
	if err != nil {
		return c, err
	}
	spt, err := azure.NewServicePrincipalToken(*oauthConfig, creds.ClientID, creds.ClientSecret, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return c, err
	}

	c.virtualMachines = compute.NewVirtualMachinesClient(creds.SubscriptionID)
	c.virtualMachines.Authorizer = spt

	c.storageAccounts = storage.NewAccountsClient(creds.SubscriptionID)
	c.storageAccounts.Authorizer = spt

	return c, nil
}

// DestroyVirtualMachine deletes a given virtual machine and the underlying storage blob.
func (c AzureClient) DestroyVirtualMachine(resourceGroup, name string) error {
	log.Printf("Destroying Virtual Machine %s in group %s\n", name, resourceGroup)
	vm, err := c.virtualMachines.Get(resourceGroup, name, "")
	if err != nil {
		return fmt.Errorf("Unable to get virtual machine: %s", err)
	}

	// Extract the OS disk information.
	vhd, _ := url.Parse(*vm.Properties.StorageProfile.OsDisk.Vhd.URI)
	storageAccount := strings.Split(vhd.Host, ".")[0]
	container := strings.Split(vhd.Path, "/")[1]
	blobName := strings.Split(vhd.Path, "/")[2]

	// We need an access key to contact the blob storage service.
	keys, err := c.storageAccounts.ListKeys(resourceGroup, storageAccount)
	if err != nil {
		return fmt.Errorf("Unable to get storage account keys: %s", err)
	}

	// Ensure we can access the storage API before we delete the VM so we don't end up in a weird state.
	sc, err := storageAPI.NewBasicClient(storageAccount, *keys.Key1)
	if err != nil {
		return fmt.Errorf("Unable to get storage client: %s", err)
	}
	blobService := sc.GetBlobService()

	// So far, everything's been successful. We can try to delete the Virtual Machine now.
	_, err = c.virtualMachines.Delete(resourceGroup, name)
	if err != nil {
		return fmt.Errorf("Unable to delete VM %s: %s", name, err)
	}

	log.Printf("VM has been deleted. Removing associated blob %s in container %s\n", blobName, container)
	// Delete the blob for the VM.
	_, err = blobService.DeleteBlobIfExists(container, blobName)
	if err != nil {
		return fmt.Errorf("Unable to delete blob %s: %s", blobName, err)
	}

	log.Printf("Blob %s has been deleted.\n", blobName)
	return nil
}
