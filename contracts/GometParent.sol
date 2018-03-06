pragma solidity ^0.4.18;

import "./OfflineMultisig.sol";

contract GometParent is OfflineMultisig {

    uint constant DUST = 0;

    // A good question here:
    // - events means what happened? <- better
    // - events means what needs to be done?

    event LogLock(uint256 epoch,  address from, uint256 value);
    event LogUnlock(address to, uint256 value);

    function GometParent(address[] _signers) 
    OfflineMultisig(_signers) public {
    }
    
    /// User calls this functions to send ETH to child chain
    function lock() payable public {
        require(msg.value > DUST);
        LogLock(epochs.length, msg.sender,msg.value);

        // PoA nodes will retrieve this event and then generates a 
        //   muliple partialExecute's for a GometChild._mint call
        // When all partialExecutes are generated, WETH is mined
        //   in GometChild
    }

    /// User calls this function to recover ETH from child chain
    function unlock(uint _epoch, bytes32 _txid, bytes _data, bytes32[] _sigs) public {
        // this should trigger _parentUnlock function, and ensures that the function is
        //  executed only and only one time
        this.fullExecuteOff(_epoch,_txid,_data,_sigs);
    }

    /* ---- multisig functions --------------------------------------- */

    function _parentUnlock(address _to, uint _value) public {
       require(msg.sender == address(this));
       _to.transfer(_value);
       LogUnlock(_to,_value);
    }


}