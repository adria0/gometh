/* global artifacts */
/* global contract */
/* global assert */

const assertFail = require("./helpers/assertFail.js");

const GometParent = artifacts.require("../contracts/GometParent.sol");
const GometChild = artifacts.require("../contracts/GometChild.sol");
const WETH = artifacts.require("../contracts/WETH.sol");

contract("GometParent", (accounts) => {

    let parent;
    let child;
    let weth;

    const {
        0: user1,
        1: user2,
        2: poa1,
        3: poa2,
        4: poa3
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
        let initialSigners = [poa1,poa2,poa3].sort()
        parent = await GometParent.new(initialSigners);
        child = await GometChild.new(initialSigners);
        weth = WETH.at(await child.weth());
    });

    it("Do the full cycle", async () => {

        // lock ethers

        const amount = web3.toWei(1,'ether')

        assert((await weth.balanceOf(user1))==0)

        let res = await parent.parentLock( { value : amount, from: user1  });
        assert(res.logs[0].event == 'LogLock');
        let lockFrom = res.logs[0].args.from  
        let lockValue = res.logs[0].args.value

        let txid = web3.sha3("txid")
        let epoch = (await child.getEpochs())-1
        let data = child.childLock.request(lockFrom,lockValue).params[0].data;

        let [v1,r1,s1] = sign(epoch,txid,data,poa1)
        await child.partialExecute(epoch,txid,data,v1,r1,s1)

        let [v2,r2,s2] = sign(epoch,txid,data,poa2)
        await child.partialExecute(epoch,txid,data,v2,r2,s2)

        assert((await weth.balanceOf(user1)).eq(amount))

    });


});