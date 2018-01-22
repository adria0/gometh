/* global artifacts */
/* global contract */
/* global assert */

const assertFail = require("./helpers/assertFail.js");

const GometBridge = artifacts.require("../contracts/GometBridge.sol");

contract("GometBridge", (accounts) => {
    let bridge;

    const {
        0: poa1,
        1: poa2,
        2: poa3,
        3: poa4,
    } = accounts;

    sign = (acc, hash) => {
        let sig = web3.eth.sign(acc, hash);
        sig = sig.substr(2, sig.length);
        let r = '0x' + sig.substr(0, 64);
        let s = '0x' + sig.substr(64, 64);
        let v = web3.toDecimal(sig.substr(128, 2)) + 27;
        return [r,s,v]
    } 

    beforeEach(async () => {
        let initial = [poa1,poa2,poa3].sort()
        bridge = await GometBridge.new([poa1,poa2,poa3].sort());
    });

    it("Add poa4 signer", async () => {

        let newsigners = [poa1,poa2,poa3,poa4].sort()

        let txid = web3.sha3("txid")
        let epoch = (await bridge.getEpochs())-1

        let data = GometBridge.setEpoch.getData(epoch+1,newsigners);
        let hash = web3.sha3(epoch,txid,data)

        let [r1,s1,v1] = sign(hash,poa1)
        bridge.childExecute(epoch,txid,data,v1,r1,s1)

        let [r2,s2,v2] = sign(hash,poa2)
        bridge.childExecute(epoch,txid,data,v2,r2,s2)

    });

});
