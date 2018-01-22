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

    const uint256hex = v => {
        return v.toString(16).padStart(64,'0')
    }

    sign = (epoch,txid, data, acc) => {

        let preimage = uint256hex(epoch)+txid.substr(2)+data.substr(2)
        let hash = web3.sha3(preimage, {encoding: 'hex'})

        var sig = web3.eth.sign(acc, hash).slice(2)

        var r = `0x${sig.slice(0, 64)}`
        var s = `0x${sig.slice(64, 128)}`
        var v = web3.toDecimal(sig.slice(128, 130)) + 27
        return [v,r,s]
    } 

    beforeEach(async () => {
        let initial = [poa1,poa2,poa3].sort()
        bridge = await GometBridge.new([poa1,poa2,poa3].sort());
    });

    it("Add new signer using partialExecute", async () => {

        let newsigners = [poa1,poa2,poa3,poa4].sort()

        let txid = web3.sha3("txid")
        let epoch = (await bridge.getEpochs())-1

        let data = bridge.changeSigners.request(epoch+1,newsigners).params[0].data;

        let [v1,r1,s1] = sign(epoch,txid,data,poa1)
        await bridge.partialExecute(epoch,txid,data,v1,r1,s1)


        let [v2,r2,s2] = sign(epoch,txid,data,poa2)
        await bridge.partialExecute(epoch,txid,data,v2,r2,s2)

        assert(await bridge.isSigner(poa4));
        assert((await bridge.getEpochs())-1==epoch+1);

    });

    it("Add new signer using fullExecute", async () => {

        let newsigners = [poa1,poa2,poa3,poa4].sort()

        let txid = web3.sha3("txid")
        let epoch = (await bridge.getEpochs())-1
        let data = bridge.changeSigners.request(epoch+1,newsigners).params[0].data;

        let [v1,r1,s1] = sign(epoch,txid,data,poa1)
        let [v2,r2,s2] = sign(epoch,txid,data,poa2)

        let sigs = ["0x"+uint256hex(v1),r1,s1,"0x"+uint256hex(v2),r2,s2]

        await bridge.fullExecute(epoch,txid,data,sigs)

        assert(await bridge.isSigner(poa4));
        assert((await bridge.getEpochs())-1==epoch+1);

    });


});
