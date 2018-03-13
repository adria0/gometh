pragma solidity ^0.4.18;

import "zeppelin-solidity/contracts/token/StandardToken.sol";
import "./PatriciaTree.sol";

contract WETH is StandardToken, PatriciaTree {

  event Log(string s);
  event StateChange(uint256 blockNo, bytes32 rootState);

  address public owner;

  function WETH(address _owner) public{
     owner = _owner;
  }

  function mint(address _to, uint256 _amount) public {
    require (msg.sender == owner);

    totalSupply = totalSupply.add(_amount);
    balances[_to] = balances[_to].add(_amount);
 
    Transfer(address(0), _to, _amount);
 
    super.insert(addr2bytes(_to),uint2bytes(balances[_to]));
    StateChange(block.number,root);
  }

  function burn(address _from, uint256 _amount) public {
    require (msg.sender == owner);

    totalSupply = totalSupply.sub(_amount);
    balances[_from] = balances[_from].sub(_amount);
    Transfer(_from, address(0), _amount);

    super.insert(addr2bytes(_from),uint2bytes(balances[_from]));
    StateChange(block.number,root);
  }

  function transfer(address _from, address _to, uint256 _amount) public {
    require (msg.sender == owner);

    balances[_from] = balances[_from].sub(_amount);
    balances[_to] = balances[_to].add(_amount);
    Transfer(_from, _to, _amount);

    super.insert(addr2bytes(_from),uint2bytes(balances[_from]));
    super.insert(addr2bytes(_to),uint2bytes(balances[_to]));
    StateChange(block.number,root);
  }

  // override transfer
  function transfer(address _to, uint256 _value) public returns (bool) {

    bool success = super.transfer(_to,_value);

    if (success && _value > 0 ) {
      super.insert(addr2bytes(msg.sender),uint2bytes(balances[msg.sender]));
      super.insert(addr2bytes(_to),uint2bytes(balances[_to]));
      StateChange(block.number,root);
    }

    return success;

  }

  // override transferFrom
  function transferFrom(address _from, address _to, uint256 _value) public returns (bool) {

    bool success = super.transferFrom(_from,_to,_value);

    if (success && _value > 0 ) {
      super.insert(addr2bytes(_from),uint2bytes(balances[_from]));
      super.insert(addr2bytes(_to),uint2bytes(balances[_to]));
      StateChange(block.number,root);
    }

    return success;

  }

  /* override PatriciaTree.insert, make it private */
  function insert(bytes, bytes) public  {
    revert();
  }

  function addr2bytes(address _v) pure internal returns (bytes) {
    bytes20 v = bytes20(_v);
    bytes memory b = new bytes(20);
    for (uint i = 0;i<20;i++) {
      b[i] = v[i];
    }
    return b;
  }
    
  function uint2bytes(uint _v) pure internal returns (bytes) {
    bytes32 v = bytes32(_v);
    bytes memory b = new bytes(32);
    for (uint i = 0;i<32;i++) {
      b[i] = v[i];
    }
    return b;
  }
    
  function bytes2addr(bytes _b)  pure internal returns (address) {
    uint256 r = 0x0;
    for (uint i = 0;i<20;i++) {
      r = r*256 + uint(_b[i]);
    }
    return address(r);
  }
    
  function bytes2uint(bytes _b) pure internal returns (uint) {
    uint256 r = 0x0;
    for (uint i = 0;i<32;i++) {
      r = r*256 + uint(_b[i]);
    }
    return uint(r);
  }
    
}