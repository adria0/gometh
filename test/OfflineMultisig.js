/* global artifacts */
/* global contract */
/* global assert */

const assertFail = require("./helpers/assertFail.js");

const OfflineMultisig = artifacts.require("../contracts/OfflineMultisig.sol");

contract("OfflineMultisig", (accounts) => {
    let multisig;

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
        return ["0x"+uint256hex(v),r,s]
    } 

    beforeEach(async () => {
        let initial = [poa1,poa2,poa3].sort()
        multisig = await OfflineMultisig.new([poa1,poa2,poa3].sort());
    });

    it("Add new signer using partialExecute", async () => {

        let newsigners = [poa1,poa2,poa3,poa4].sort()

        let txid = web3.sha3("txid")
        let epoch = (await multisig.getEpochs())-1

        let data = multisig._changeSigners.request(epoch+1,newsigners).params[0].data;

        await multisig.partialExecuteOff(epoch,txid,data,sign(epoch,txid,data,poa1))
        await multisig.partialExecuteOff(epoch,txid,data,sign(epoch,txid,data,poa2))

        assert(await multisig.isSigner(poa4));
        assert((await multisig.getEpochs())-1==epoch+1);

    });

    it("Add new signer using fullExecute", async () => {

        let newsigners = [poa1,poa2,poa3,poa4].sort()

        let txid = web3.sha3("txid")
        let epoch = (await multisig.getEpochs())-1
        let data = multisig._changeSigners.request(epoch+1,newsigners).params[0].data;

        let sigs = sign(epoch,txid,data,poa1).concat(sign(epoch,txid,data,poa2))
        await multisig.fullExecuteOff(epoch,txid,data,sigs)

        assert(await multisig.isSigner(poa4));
        assert((await multisig.getEpochs())-1==epoch+1);

    });

    it("Add new signer in one, collect signatures and update into another", async () => {
        
        let newsigners = [poa1,poa2,poa3,poa4].sort()

        let txid = web3.sha3("txid")
        let epoch = (await multisig.getEpochs())-1

        let data = multisig._changeSigners.request(epoch+1,newsigners).params[0].data;

        await multisig.partialExecuteOff(epoch,txid,data,sign(epoch,txid,data,poa1))
        await multisig.partialExecuteOff(epoch,txid,data,sign(epoch,txid,data,poa2))

        assert(await multisig.isSigner(poa4));
        assert((await multisig.getEpochs())-1==epoch+1);

        let [csdata, cssigs] = await multisig.getSignatures(epoch,txid)

        let multisig2 = await OfflineMultisig.new([poa1,poa2,poa3].sort());
        await multisig2.fullExecuteOff(epoch,txid,csdata,cssigs)
        assert(await multisig2.isSigner(poa4));
        
    });

});
