pragma solidity ^0.4.18;

import "./GometMultisig.sol";

contract GometParent is GometMultisig {

    event LogLock(address from, uint value);

    function GometParent(address[] _signers) 
    GometMultisig(_signers) public {
    }
    
    function parentLock() payable public {
        require(msg.value > 0);
        LogLock(msg.sender,msg.value);
    }
    
    function parentUnlock(address _to, uint _value) public {
       require(msg.sender == address(this));
       _to.transfer(_value);
    }
    
}