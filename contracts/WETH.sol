pragma solidity ^0.4.18;

import "zeppelin-solidity/contracts/token/StandardToken.sol";

contract WETH is StandardToken {

  address owner;

  function WETH() public{
     owner = msg.sender;
  }

  function mint(address _to, uint256 _amount) public {
    require (msg.sender == owner);

    totalSupply = totalSupply.add(_amount);
    balances[_to] = balances[_to].add(_amount);
    Transfer(address(0), _to, _amount);
  }

  function burn(address _from, uint256 _amount) public {
    require (msg.sender == owner);

    totalSupply = totalSupply.sub(_amount);
    balances[_from] = balances[_from].sub(_amount);
    Transfer(_from, address(0), _amount);
  }

}