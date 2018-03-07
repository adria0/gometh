pragma solidity ^0.4.18;

// got from https://github.com/chriseth/patricia-trie

library D {
    struct Label {
        bytes32 data;
        uint length;
    }
    struct Edge {
        bytes32 node;
        Label label;
    }
    struct Node {
        Edge[2] children;
    }
}

library Utils {
    /// Returns a label containing the longest common prefix of `check` and `label`
    /// and a label consisting of the remaining part of `label`.
    function splitCommonPrefix(D.Label label, D.Label check) pure internal returns (D.Label prefix, D.Label labelSuffix) {
        return splitAt(label, commonPrefix(check, label));
    }
    /// Splits the label at the given position and returns prefix and suffix,
    /// i.e. prefix.length == pos and prefix.data . suffix.data == l.data.
    function splitAt(D.Label l, uint pos) pure internal returns (D.Label prefix, D.Label suffix) {
        require(pos <= l.length && pos <= 256);
        prefix.length = pos;
        if (pos == 0) {
            prefix.data = bytes32(0);
        } else {
            prefix.data = l.data & ~bytes32((uint(1) << (256 - pos)) - 1);
        }
        suffix.length = l.length - pos;
        suffix.data = l.data << pos;
    }
    /// Returns the length of the longest common prefix of the two labels.
    function commonPrefix(D.Label a, D.Label b) pure  internal returns (uint prefix) {
        uint length = a.length < b.length ? a.length : b.length;
        // TODO: This could actually use a "highestBitSet" helper
        uint diff = uint(a.data ^ b.data);
        uint mask = 1 << 255;
        for (; prefix < length; prefix++)
        {
            if ((mask & diff) != 0)
                break;
            diff += diff;
        }
    }
    /// Returns the result of removing a prefix of length `prefix` bits from the
    /// given label (i.e. shifting its data to the left).
    function removePrefix(D.Label l, uint prefix) pure internal returns (D.Label r) {
        require(prefix <= l.length);
        r.length = l.length - prefix;
        r.data = l.data << prefix;
    }
    /// Removes the first bit from a label and returns the bit and a
    /// label containing the rest of the label (i.e. shifted to the left).
    function chopFirstBit(D.Label l) pure internal returns (uint firstBit, D.Label tail) {
        require(l.length > 0);
        return (uint(l.data >> 255), D.Label(l.data << 1, l.length - 1));
    }
    /// Returns the first bit set in the bitfield, where the 0th bit
    /// is the least significant.
    /// Throws if bitfield is zero.
    /// More efficient the smaller the result is.
    function lowestBitSet(uint bitfield) pure internal returns (uint bit) {
        require(bitfield != 0);
        bytes32 bitfieldBytes = bytes32(bitfield);
        // First, find the lowest byte set
        for (uint byteSet = 0; byteSet < 32; byteSet++) {
            if (bitfieldBytes[31 - byteSet] != 0)
                break;
        }
        uint singleByte = uint(uint8(bitfieldBytes[31 - byteSet]));
        uint mask = 1;
        for (bit = 0; bit < 256; bit ++) {
            if ((singleByte & mask) != 0)
                return 8 * byteSet + bit;
            mask += mask;
        }
        assert(false);
        return 0;
    }
    /// Returns the value of the `bit`th bit inside `bitfield`, where
    /// the least significant is the 0th bit.
    function bitSet(uint bitfield, uint bit) pure internal returns (uint) {
        return (bitfield & (uint(1) << bit)) != 0 ? 1 : 0;
    }
}

contract PatriciaTree {

 	address owner;

	function PatriciaTree() public { // added
		owner = msg.sender;
	}

    // Mapping of hash of key to value
    mapping (bytes32 => bytes) values;

    // Particia tree nodes (hash to decoded contents)
    mapping (bytes32 => D.Node) nodes;
    // The current root hash, keccak256(node(path_M('')), path_M(''))
    bytes32 public root;
    D.Edge rootEdge;

    function getNode(bytes32 hash) public constant returns (uint, bytes32, bytes32, uint, bytes32, bytes32) {
        var n = nodes[hash];
        return (
            n.children[0].label.length, n.children[0].label.data, n.children[0].node,
            n.children[1].label.length, n.children[1].label.data, n.children[1].node
        );
    }

    function getRootEdge() public  constant returns (uint, bytes32, bytes32) {
        return (rootEdge.label.length, rootEdge.label.data, rootEdge.node);
    }
    
    function edgeHash(D.Edge e) pure internal returns (bytes32) {
        return keccak256(e.node, e.label.length, e.label.data);
    }
    
    // Returns the hash of the encoding of a node.
    function hash(D.Node memory n) pure internal returns (bytes32) {
        return keccak256(edgeHash(n.children[0]), edgeHash(n.children[1]));
    }
    
    // Returns the Merkle-proof for the given key
    // Proof format should be:
    //  - uint branchMask - bitmask with high bits at the positions in the key
    //                    where we have branch nodes (bit in key denotes direction)
    //  - bytes32[] hashes - hashes of sibling edges
    function getProof(bytes key) public  view  returns (uint branchMask, bytes32[] _siblings) {
        D.Label memory k = D.Label(keccak256(key), 256);
        D.Edge memory e = rootEdge;
        bytes32[256] memory siblings;
        uint length;
        uint numSiblings;
        while (true) {
            var (prefix, suffix) = Utils.splitCommonPrefix(k, e.label);
            require(prefix.length == e.label.length);
            if (suffix.length == 0) {
                // Found it
                break;
            }
            length += prefix.length;
            branchMask |= uint(1) << (255 - length);
            length += 1;
            var (head, tail) = Utils.chopFirstBit(suffix);
            siblings[numSiblings++] = edgeHash(nodes[e.node].children[1 - head]);
            e = nodes[e.node].children[head];
            k = tail;
        }
        if (numSiblings > 0)
        {
            _siblings = new bytes32[](numSiblings);
            for (uint i = 0; i < numSiblings; i++)
                _siblings[i] = siblings[i];
        }
    }

    function verifyProof(bytes32 rootHash, bytes key, bytes value, uint branchMask, bytes32[] siblings) public pure {
        D.Label memory k = D.Label(keccak256(key), 256);
        D.Edge memory e;
        e.node = keccak256(value);
        for (uint i = 0; branchMask != 0; i++) {
            uint bitSet = Utils.lowestBitSet(branchMask);
            branchMask &= ~(uint(1) << bitSet);
            (k, e.label) = Utils.splitAt(k, 255 - bitSet);
            uint bit;
            (bit, e.label) = Utils.chopFirstBit(e.label);
            bytes32[2] memory edgeHashes;
            edgeHashes[bit] = edgeHash(e);
            edgeHashes[1 - bit] = siblings[siblings.length - i - 1];
            e.node = keccak256(edgeHashes);
        }
        e.label = k;
        require(rootHash == edgeHash(e));
    }
    
    // TODO also return the proof
    function insert(bytes key, bytes value) public  {

        D.Label memory k = D.Label(keccak256(key), 256);
        bytes32 valueHash = keccak256(value);
        values[valueHash] = value;
        // keys.push(key);
        D.Edge memory e;
        if (rootEdge.node == 0 && rootEdge.label.length == 0)
        {
            // Empty Trie
            e.label = k;
            e.node = valueHash;
        }
        else
        {
            e = insertAtEdge(rootEdge, k, valueHash);
        }
        root = edgeHash(e);
        rootEdge = e;
    }
    
    function insertAtNode(bytes32 nodeHash, D.Label key, bytes32 value) internal returns (bytes32) {
        require(key.length > 1);
        D.Node memory n = nodes[nodeHash];
        var (head, tail) = Utils.chopFirstBit(key);
        n.children[head] = insertAtEdge(n.children[head], tail, value);
        return replaceNode(nodeHash, n);
    }
    
    function insertAtEdge(D.Edge e, D.Label key, bytes32 value) internal returns (D.Edge) {
        require(key.length >= e.label.length);
        var (prefix, suffix) = Utils.splitCommonPrefix(key, e.label);
        bytes32 newNodeHash;
        if (suffix.length == 0) {
            // Full match with the key, update operation
            newNodeHash = value;
        } else if (prefix.length >= e.label.length) {
            // Partial match, just follow the path
            newNodeHash = insertAtNode(e.node, suffix, value);
        } else {
            // Mismatch, so let us create a new branch node.
            var (head, tail) = Utils.chopFirstBit(suffix);
            D.Node memory branchNode;
            branchNode.children[head] = D.Edge(value, tail);
            branchNode.children[1 - head] = D.Edge(e.node, Utils.removePrefix(e.label, prefix.length + 1));
            newNodeHash = insertNode(branchNode);
        }
        return D.Edge(newNodeHash, prefix);
    }

    function insertNode(D.Node memory n) internal returns (bytes32 newHash) {
        bytes32 h = hash(n);
        nodes[h].children[0] = n.children[0];
        nodes[h].children[1] = n.children[1];
        return h;
    }

    function replaceNode(bytes32 oldHash, D.Node memory n) internal returns (bytes32 newHash) {
        delete nodes[oldHash];
        return insertNode(n);
    }
}