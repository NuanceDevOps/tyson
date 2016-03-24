# Tyson

A small utility to destroy machines in Azure. Azure lacks a native feature to delete a VM and its underlying storage. This makes it impossible to destroy and re-create using ARM templates automatically.

## Usage

```bash
Usage of tyson:
  -credentials.file string
    	Specify the JSON file with the Azure credentials. (default "~/.azure/credentials.json")
  -f	Do not prompt user before destroying.
  -random
    	Randomly select a machine to destroy.
  -regex string
    	Regex to use during random selection. (default ".*")
  -resource.group string
    	Resource group of virtual machine. If random is selected, the search will be limited to this group.
  -vm.name string
    	Name of virtual machine to destroy. Not used if random is selected.
```

## Binary releases

Pre-compiled versions may be found in the [release section](https://github.com/iamseth/tyson/releases).
