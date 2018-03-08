pragma solidity ^0.4.18;

import "./OfflineMultisig.sol";
import "./WETH.sol";

contract GometChild is OfflineMultisig {

    event LogBurn(address from, uint value);
    event LogMintMultisigned(address to, uint value);
    event LogStateChangeMultisigned(uint256 blockNo, bytes32 rootState);

    WETH public weth;

    function GometChild(address[] _signers) public 
    OfflineMultisig(_signers) {
        weth = new WETH();
    }

    function burn(uint _amount) public {
       weth.burn(msg.sender,_amount);
       LogBurn(msg.sender,_amount);
    }

    function toLocalEther(uint _amount) public {
      address[] storage signers = epochs[epochs.length-1];
      uint share = _amount / signers.length;

      // split the amount between the current PoAs
      for (uint i=0;i<signers.length;i++) {
        weth.transfer(msg.sender,signers[i],share);
      }

      // send the ethers to the current signers
      msg.sender.transfer(share*signers.length);
    }

    /* ---- multisig functions --------------------------------------- */

    function _mint(address _to, uint _amount) public {
       require(msg.sender == address(this));

       LogMintMultisigned(_to,_amount);
       weth.mint(_to,_amount);

       // send a litte of ether to call toLocalEther
       if (_to.balance < 0.01 ether ) {
         _to.transfer(0.01 ether - _to.balance);
       }
    }

    // this function is called via partialExecuteOff, this means that all
    //     offline transactions will be available, this means that this calls
    //     can be called also into the parent chain

    function _statechangemultisigned(uint256 blockNo, bytes32 rootState) public {
       require(msg.sender == address(this));      
       LogStateChangeMultisigned(blockNo, rootState);
    }

}
