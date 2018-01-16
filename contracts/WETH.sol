pragma solidity ^0.4.19;

import "zeppelin-solidity/contracts/token/StandardToken.sol";
import "./BurnCallback.sol";

contract WETH {
  
  address constant public BURNADDR = 0x1;
  BurnCallback burnCallback;

  function WETH(address _burnCallback) {
     burnCallback = BurnCallback(_burnCallback);
  }

  function mint(address _to, uint256 _amount) onlyOwner {
    totalSupply = totalSupply.add(_amount);
    balances[_to] = balances[_to].add(_amount);
    Transfer(address(0), _to, _amount);
    return true;
  }

  // override transfer function
  function transfer(address _to, uint256 _value) public returns (bool) {
      super.transfer(_to,_value);
      if (_to == BURNADDR) {
         burnCallback(msg.sender,_value);
      }
  }
    
}