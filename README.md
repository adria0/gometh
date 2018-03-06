# gometh

## Setup

- A Rinkeby child-chain PoA is created with signers { PoA1, PoA2, ... }
- The GometParent contract is deployed in the main net and initialized with PoA signers
- The GometChild contract is deployed in the main net and initialized with PoA signers

## Sending ethers from the Parent -> Child

- parentContract.lock() is called
  - this generates a LogLock() event
  - child chain nodes will check this event and will generate
    a amount into the child chain using mutis



## Troubleshooting






Change signers

- For each each PoA
  - generates a offline signature to execute _changeSigners
  - executes partialExecute() on child chain for this _changeSigners
  - waits if the LogSignersChanged is triggered, when triggered
    - changes the configuration of the authorities
    - generates an LogPoASuccessConfigChange (optional)


Parent                                          Child
---------                                       ----------
partialExecute(_changeSigners,[], from{Poa1})
partialExecute(_changeSigners,[], from{Poa2})   partialExecute(_changeSigners,[], from{Poa2})
partialExecute(_changeSigners,[], from{Poa3})   partialExecute(_changeSigners,[], from{Poa3})
Event LogSignersChanged                         Event LogSignersChanged
                                                Event LogPoASuccessConfigChange(address poa1)*
                                                Event LogPoASuccessConfigChange(address poa2)*
                                                Event LogPoASuccessConfigChange(address poa3)*

* not mandatory


ETH <-> WETH GENERATION
                                                
                          

Parent         Child
--------- ------------

lock() -------> mint()
                generateEth()
unlock() <----  burn()

each time a voucher is created, the root hash of the WETH contract is also added


--------------------------------------
RECOVERY
--------------------------------------
Parent contracts needs to be "kept alive" with a multisignature of 
- WETH state root 
- child chain block

If a ping is not recieved in XXX blocks
  - lock ethers to child chain is locked

If a ping is not recieved in YYY > XXX blocks
  - anybody can call the globalSettelment()
  - a global settelment is done at 

When globalSettelment it executed anybody is able to:
- 1) Go to the child chain data, and query a proof from a block:
     child.queryProof(root, myaddress, { block : blockNo } ) -> a_bunch_of_data
- 2) Go to the parent chain and process the data:      
     parent.processProof(myaddress, a_bunch_of_data)

