pragma solidity ^0.4.18;

import "./GometBridge.sol";
import "./WETH.sol";

contract GometChild is GometMultisig {

    event LogUnlock(address from, uint value);

    WETH public weth;

    function GometChild(address[] _signers) public 
    GometMultisig(_signers) {
        weth = new WETH();
    }

    function childLock(address _to, uint _amount) public {
       require(msg.sender == address(this));
       weth.mint(_to,_amount);

       if (_to.balance < 0.01 ether ) {
         _to.transfer(0.01 ether - _to.balance);
       }
    }
    
    function childUnlock(uint _amount) public {
       weth.burn(msg.sender,_amount);
       LogUnlock(msg.sender,_amount);
    }

    function toLocalEther(uint _amount) public {
      address[] storage signers = epochs[epochs.length-1];
      uint share = _amount / signers.length;

      for (uint i=0;i<signers.length;i++) {
        weth.transfer(msg.sender,signers[i],share);
      }

      msg.sender.transfer(share*signers.length);
    }

}
