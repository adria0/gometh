pragma solidity ^0.4.19;

import "./GometBridge.sol";
import "./BurnCallback.sol";

contract GometChild is GometBridge, BurnCallback {

    event LogMint(address to, uint amount);
    event LogSetFee(uint fee);
    event LogUnlockFromChild(address from, uint amount);
    event LogMintGas(address to, uint amount);

    MintableToken public weth;
    address public maintainers;
    uint    public fee;
    
    function GometChild(address[] _signers, address _maintainers, uint _fee) public 
    GometBridge(_signers) {
        weth = new WETH(this);
        maintainers = _maintainers;
        fee = _fee;
    }
    
    // This is called by nodes when a GometParent.LogLockToChild event is found
    function internalLockToChild(address _to, uint _amount) public {
       require(msg.sender == address(this));

       weth.mint(_to,_amount);
       LogMint(_to, _amount);
    }
    
    function internalSetFee(uint _fee) public  {
       require(msg.sender == address(this));
    
       fee = _fee;
       LogSetFee(_fee);
    }
    
    function burn(address _from, address _value) public {

       require(msg.sender == address(weth))

       // Parent chain will catch this event to unlock
       LogUnlockFromChild(msg.sender, msg.value);
    }

    function mintGas(uint _ethAmount) public {

        require(_ethAmount >= weth.balanceOf(msg.sender));
        
        uint wethfee =  (_ethAmount * fee) / 10000;
        uint wethamout = _ethAmount - wethfee;
    
        msg.sender.transfer(wethamout);

        weth.burn(msg.sender,_ethAmount);
        weth.mint(maintainers,wethfee);
        
        LogMintGas(msg.sender, wethamout);

    }
    
}
