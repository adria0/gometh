pragma solidity ^0.4.18;

contract GometParentTest {

    event LogLock(address from,uint256 value);

    /// User calls this functions to send ETH to child chain
    function lock() payable public {
        LogLock(msg.sender,msg.value);

        // PoA nodes will retrieve this event and then generates a 
        //   muliple partialExecute's for a GometChild._mint call
        // When all partialExecutes are generated, WETH is mined
        //   in GometChild
    }

}