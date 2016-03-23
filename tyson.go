package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	credentials   = flag.String("credentials.file", getHomeDirectory()+"/.azure/credentials.json", "Specify the JSON file with the Azure credentials.")
	random        = flag.Bool("random", false, "Randomly select a machine to destroy. Not yet implemented")
	force         = flag.Bool("f", false, "Do not prompt user before destroying.")
	resourceGroup = flag.String("resource.group", "", "Resource group of virtual machine.")
	vmName        = flag.String("vm.name", "", "Name of virtual machine to destroy.")
)

func init() {
	flag.Parse()

	if *random {
		log.Fatal("Not yet implemented.")
	}
	if !*random && *resourceGroup == "" {
		log.Fatal("A resource group is required if random selection is disabled.")
	}

	if !*random && *vmName == "" {
		log.Fatal("A virtual machine name is required if random selection is disabled.")
	}

	if !*force {
		m := fmt.Sprintf("Are you sure you want to delete %s in resource group %s? ", *vmName, *resourceGroup)
		proceed := promptUser(m)
		if !proceed {
			os.Exit(0)
		}
	}
}

func main() {
	creds := NewCredentialsFromFile(*credentials)
	client, err := NewAzureClient(creds)
	if err != nil {
		log.Fatalf("Unable to create an Azure client: %s", err)
	}

	err = client.DestroyVirtualMachine(*resourceGroup, *vmName)
	if err != nil {
		log.Fatalf("I was unable to destroy the virtual machine. %s", err)
	}

}
