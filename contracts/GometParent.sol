pragma solidity ^0.4.19;

import "./GometBridge.sol";


contract GometParent is GometBridge {
    
    event LogLockToChild(address from, uint value);
    event LogUnlockFromChild(address to, uint value);
    
    function GometParent(address[] _signers) 
    GometBridge(_signers) public
    {
    }
    
    function lockToChild() public {
        LogLockToChild(msg.sender,msg.value);
    }
    
    function internalUnlockFromChild(address _to, uint _value) public {
       require(msg.sender == address(this));
       _to.transfer(_value);
       LogUnlockFromChild(_to,_value);
    }
    
}