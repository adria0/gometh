pragma solidity ^0.4.18;

contract GometBridge {

    bytes constant web3SignaturePrefix = "\x19Ethereum Signed Message:\n32";

    address[][] public epochs;
    struct Transaction {
        uint count;
        bool executed;
        mapping (address=>bool)  approved;
    }
    mapping (bytes32=>Transaction) public transactions;

    function getEpochs() public view returns (uint) {
       return epochs.length;
    }

    function GometBridge(address[] _signers) public {
      require(checkSignersOrder(_signers));
      uint epoch = epochs.length++;
      epochs[epoch].length = _signers.length;
      for (uint i=0;i<_signers.length;i++) {
          epochs[epoch][i] = _signers[i];
      }
    }

    function checkSignersOrder(address[] _signers) internal pure returns (bool) {
      for (uint i=0;i<_signers.length;i++) {
         if (i>0 && uint(_signers[i-1])>=uint(_signers[i])) {
            return false;
         }
      }
      return true;
    }
    
    function isSigner( address _addr) public view returns (bool) {
        uint epoch = epochs.length-1;
        for (uint i=0;i<epochs[epoch].length;i++) {
           if (epochs[epoch][i]==_addr) {
              return true;
            }
        }
        return false;      
     }
    
    function verifyMultiSignature(uint _epoch, bytes32 _hash, bytes32[] _sigs) view public
    returns (bool) {

        uint signerNo=0;
        
        address[] storage signers = epochs[epochs.length-1];
        if ( _epoch != 0 ) {
            signers = epochs[_epoch];
        }
        
        for (uint i=0;i<_sigs.length;i+=3) {
          
          // retrieve the signer
          
          uint8 v = uint8(uint256(_sigs[i]));
          bytes32 r = _sigs[i+1];
          bytes32 s = _sigs[i+2];
          address signer = ecrecover(_hash,v,r,s); 

          // check that this signer exists in the current signer list
          
          while (signerNo<signers.length && signers[signerNo]!=signer) {
              signerNo++;
          }
          if (signerNo>=signers.length) {
              return false;
          }
          
          // jump to the next signer, to avoid duplicates
          
          signerNo++;
    
        }
        
        return true;
     }
    
    
    // parent chain execution
    function fullExecute(uint _epoch, bytes32 _txid, bytes _data, bytes32[] _sigs) public {
        
        bytes32 hash = keccak256(_epoch,_txid,_data);
        bytes32 prefixedHash = keccak256(web3SignaturePrefix, hash);

        require(verifyMultiSignature(_epoch,prefixedHash,_sigs));
        require(!transactions[hash].executed);
        require(this.call(_data));
        transactions[hash].executed = true;
    }

    // child chain execution
    function partialExecute(uint256 _epoch, bytes32 _txid, bytes _data, uint8 _v, bytes32 _r, bytes32 _s) public {

        address[] storage signers = epochs[epochs.length-1];

        bytes32 hash = keccak256(_epoch,_txid,_data);
        bytes32 prefixedHash = keccak256(web3SignaturePrefix, hash);

        address signer = ecrecover(prefixedHash,_v,_r,_s);

        require (isSigner(signer));

        require (!transactions[hash].approved[signer]);
        require (!transactions[hash].executed);

        transactions[hash].count++;
        transactions[hash].approved[signer]=true;
          
        bool quorum = transactions[hash].count == (2 * signers.length) /3;

        if (quorum) {
            require(this.call(_data));
            transactions[hash].executed=true;
        }
        
    }
    
    /* ---- multisig ------------------------------------------------ */

    event LogSignersChanged(uint epoch, address[] signers);

    function changeSigners(uint _epoch, address[] _signers) public {
        
        require (msg.sender == address(this));
        
        require (_epoch == epochs.length);
        require (checkSignersOrder(_signers));
      
          uint epoch = epochs.length++;
          epochs[epoch].length = _signers.length;
          for (uint i=0;i<_signers.length;i++) {
              epochs[epoch][i] = _signers[i];
          }

        LogSignersChanged(_epoch,_signers);
    } 
    
}

