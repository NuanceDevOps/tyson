package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"

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

// ListAllVirtualMachines returns a slice of all virtual machines for a subscription
func (c AzureClient) ListAllVirtualMachines() ([]compute.VirtualMachine, error) {
	var machines []compute.VirtualMachine
	result, err := c.virtualMachines.ListAll()
	if err != nil {
		return machines, fmt.Errorf("could not list virtual machines: %s", err)
	}
	machines = append(machines, *result.Value...)

	// If we still have results, keep going until we have no more.
	for result.NextLink != nil {
		result, err = c.virtualMachines.ListAllNextResults(result)
		if err != nil {
			return machines, fmt.Errorf("could not list virtual machines: %s", err)
		}
		machines = append(machines, *result.Value...)
	}
	return machines, nil
}

// ListVirtualMachines returns a slice of virtual machines for a particular resource group
func (c AzureClient) ListVirtualMachines(resourceGroupName string) ([]compute.VirtualMachine, error) {
	result, err := c.virtualMachines.List(resourceGroupName)
	if err != nil {
		return *result.Value, err
	}
	return *result.Value, nil
}

// RandomVirtualMachine selects a random virtual machine based on regex string provided.
// The group param will restrict the search to a particular resource group which may be "" for all groups.
func (c AzureClient) RandomVirtualMachine(regex, group string) (compute.VirtualMachine, error) {
	log.Println("Finding a random target to destroy.")
	var machines []compute.VirtualMachine
	var err error
	// If we don't have a group, get all machines
	if group == "" {
		machines, err = c.ListAllVirtualMachines()
		if err != nil {
			return compute.VirtualMachine{}, err
		}
	} else {
		machines, err = c.ListVirtualMachines(group)
		if err != nil {
			return compute.VirtualMachine{}, err
		}
	}

	// Shuffle the slice so we get a random-ish order
	rand.Seed(time.Now().UTC().UnixNano())
	for i := range machines {
		j := rand.Intn(i + 1)
		machines[i], machines[j] = machines[j], machines[i]
	}

	// We should be in a fairly random order so just return first match.
	for _, machine := range machines {
		matched, err := regexp.MatchString(regex, *machine.Name)
		if err != nil {
			return compute.VirtualMachine{}, err
		}
		if matched {
			log.Printf("Found target %s.", *machine.Name)
			return machine, nil
		}
	}
	return compute.VirtualMachine{}, fmt.Errorf("No VM found matching regex '%s' and/or in group '%s'", regex, group)
}
