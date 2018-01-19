pragma solidity ^0.4.18;

import "./GometBridge.sol";
import "./WETH.sol";

contract GometChild is GometBridge {

    event LogUnlock(address from, uint value);

    WETH    public weth;

    function GometChild(address[] _signers) public 
    GometBridge(_signers) {
        weth = new WETH();
    }

    function multisigChildLock(address _to, uint _amount) public {
       require(msg.sender == address(this));
       weth.mint(_to,_amount);
    }
    
    function unlock(uint _amount) public {
       weth.burn(msg.sender,_amount);
       LogUnlock(msg.sender,_amount);
    }
}

