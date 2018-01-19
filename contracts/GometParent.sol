pragma solidity ^0.4.18;

import "./GometBridge.sol";

contract GometParent is GometBridge {

    event LogLock(address from, uint value);

    function GometParent(address[] _signers) 
    GometBridge(_signers) public {
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