pragma solidity ^0.4.19;

contract GometBridge {
    
    event LogTransactionCreated(bytes32 hash, uint nonce, bytes data);
    event LogTransactionApproved(bytes32 hash, uint nonce, bytes data);
    event LogTransactionDiscarded(bytes32 hash);
    event LogTransactionRevoked(bytes32 hash);
    event LogTransactionExecuted(bytes32 hash, uint nonce, bytes data, bytes32[] sigs);
    
    function GometBridge(address[] _signers) public {
      require(checkSignersOrder(_signers));
      
      signers.length = _signers.length;
      for (uint i=0;i<_signers.length;i++) {
          signers[i] = _signers[i];
      }
    }
    
    function checkSignersOrder(address[] _signers) internal pure returns (bool) {
      for (uint i=0;i<_signers.length;i++) {
         if (i>0 && uint(_signers[i-1])<uint(_signers[i])) {
            return false;
         }
      }
    }
    
    address[] public signers;
    
    struct Signature {
       bool    exists;
       uint8   v;
       bytes32 r;
       bytes32 s;
    }

    struct Transaction {
        bool    exists;
        bool    executed;
        uint    blockstart;
        uint    nonce;
        bytes   data;
        uint    count;
        mapping (address=>Signature) sigs;
    }
    
    mapping (bytes32 => Transaction) public trns;

    function verifySignature(bytes32 _hash, bytes32[] _sigs) view public
    returns (bool) {

        uint signerNo=0;
        
        for (uint i=0;i<_sigs.length;i+=3) {
          
          // retrieve the signer
          
          uint8 v = uint8(_sigs[i][0]);
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

    function isSigner( address _addr) public view returns (bool) {
    
        for (uint i=0;i<signers.length;i++) {
           if (signers[i]==_addr) {
              return true;
            }
        }
        return false;      
     }
 
    
    function approveTransactionHash(uint _nonce, bytes _data) public view returns (bytes32) {
        return keccak256("approveTransaction:",keccak256(_nonce,_data));
    }
    
    function revokeTransactionHash(bytes32 _hash) public view returns (bytes32) {
        return keccak256("approveTransaction:",_hash);
    }

    function createTransaction(uint _nonce, bytes _data, uint8 _v, bytes32 _r, bytes32 _s) public {
        require(isSigner(msg.sender));
        
        bytes32 hash = keccak256(_nonce,_data);
        
        trns[hash].exists = true;
        trns[hash].nonce = _nonce;
        trns[hash].executed = false;
        trns[hash].data = _data;
        trns[hash].blockstart = block.number + 50;
        
        LogTransactionCreated(hash, _nonce, _data);
        
        approveTransaction(hash,_v,_r,_s);
    }
    
    /// approve a request
    function approveTransaction(bytes32 _hash, uint8 _v, bytes32 _r, bytes32 _s) public {
    
      address signer = ecrecover(keccak256("approveTransaction:",_hash),_v,_r,_s);

      require(isSigner(signer));
      require(!trns[_hash].sigs[signer].exists);

      trns[_hash].sigs[signer].exists = true;
      trns[_hash].sigs[signer].v = _v;
      trns[_hash].sigs[signer].r = _r;
      trns[_hash].sigs[signer].s = _s;
      trns[_hash].count++;

      if ((2 * trns[_hash].count) /3  >= signers.length) {
          if (trns[_hash].blockstart >= block.number) {
              executeTransaction(_hash);
          }
      }
    }
    
    function revokeTransaction(bytes32 _hash,  uint8 _v, bytes32 _r, bytes32 _s) public {
      address signer = ecrecover(keccak256("revokeTransaction:",_hash),_v,_r,_s);
      require(isSigner(signer));
      
      delete trns[_hash];
      LogTransactionRevoked(_hash);
    }
    
    event LogDiscardedSignature(bytes32 _hash);
    
    function executeTransaction(bytes32 _hash) public {
     
      require(trns[_hash].exists);
      require(!trns[_hash].executed);
      require(trns[_hash].blockstart >= block.number);

      // -- pack signatures 
      bytes32[] memory sigs=new bytes32[](3*trns[_hash].count);
      uint offset = 0;
      for (uint i=0;i<signers.length;i++) {
         Signature memory sig = trns[_hash].sigs[signers[i]]; 
         if (sig.v != 0)  {
             sigs[offset++]=bytes32(sig.v);
             sigs[offset++]=sig.r;
             sigs[offset++]=sig.s;
         }
      }
      
      // -- at this point check all signatures are processed,
      //    if in some point a signer is removed, this means that
      //    this transaction should be discarded
      
      if (offset < 3*trns[_hash].count) {
          LogTransactionDiscarded(_hash);
          delete trns[_hash];
          return;
      }

      LogTransactionExecuted(_hash, trns[_hash].nonce, trns[_hash].data, sigs);

      require(this.call(trns[_hash].data));
      trns[_hash].executed = true;
      
    }

    function executeTransaction(uint _nonce,  bytes _data, bytes32[] _sigs) public {
        
        bytes32 hash = keccak256(_nonce,_data);
        require(!trns[hash].exists);

        trns[hash].exists = true;
        require(verifySignature(hash,_sigs));
        
        require(this.call(_data));

    }
    function internalChangeSigners( address[] _signers) public {
      
       require(msg.sender == address(this));
       require(checkSignersOrder(_signers));
      
       signers.length = _signers.length;
       for (uint i=0;i<_signers.length;i++) {
           signers[i] = _signers[i];
       }
       
    }
     
  
}

