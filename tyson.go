package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
)

var (
	credentials   = flag.String("credentials.file", getHomeDirectory()+"/.azure/credentials.json", "Specify the JSON file with the Azure credentials.")
	random        = flag.Bool("random", false, "Randomly select a machine to destroy.")
	regex         = flag.String("regex", ".*", "Regex to use during random selection.")
	force         = flag.Bool("f", false, "Do not prompt user before destroying.")
	resourceGroup = flag.String("resource.group", "", "Resource group of virtual machine.")
	vmName        = flag.String("vm.name", "", "Name of virtual machine to destroy.")
)

func main() {
	flag.Parse()

	if !*random && *resourceGroup == "" {
		log.Fatal("A resource group is required if random selection is disabled.")
	}

	if !*random && *vmName == "" {
		log.Fatal("A virtual machine name is required if random selection is disabled.")
	}

	creds := NewCredentialsFromFile(*credentials)
	client, err := NewAzureClient(creds)
	if err != nil {
		log.Fatalf("Unable to create an Azure client: %s", err)
	}

	if *random {
		log.Println("Finding a random target to destroy.")
		var vm compute.VirtualMachine
		vm, err = client.RandomVirtualMachine(*regex, *resourceGroup)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Found target %s.", *vm.Name)
		vmName = vm.Name
		*resourceGroup = strings.Split(*vm.ID, "/")[4]
	}

	if !*force {
		m := fmt.Sprintf("Are you sure you want to delete %s in resource group %s? ", *vmName, *resourceGroup)
		proceed := promptUser(m)
		if !proceed {
			os.Exit(0)
		}
	}

	err = client.DestroyVirtualMachine(*resourceGroup, *vmName)
	if err != nil {
		log.Fatalf("I was unable to destroy the virtual machine. %s", err)
	}

}
