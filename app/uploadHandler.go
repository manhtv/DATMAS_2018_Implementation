package app

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	conf "github.com/racin/DATMAS_2018_Implementation/configuration"
	"github.com/racin/DATMAS_2018_Implementation/crypto"
	"fmt"
	"github.com/racin/DATMAS_2018_Implementation/types"
	"time"
	"sync"
)

func (app *Application) StartUploadHandler(){
	http.HandleFunc(conf.AppConfig().UploadEndpoint, app.UploadHandler)
	if err := http.ListenAndServe(app.uploadAddr, nil); err != nil {
		panic("Error setting up upload handler. Error: " + err.Error())
	}
}

func writeUploadResponse(w *http.ResponseWriter, codeType types.CodeType, message string){
	//json.NewEncoder(*w).Encode(&types.ResponseUpload{Message:message, Codetype:codeType})
	byteArr, _ := json.Marshal(&types.ResponseUpload{Message:message, Codetype:codeType})
	fmt.Fprintf(*w, "%s", byteArr)
}

func (app *Application) UploadHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Uploadhandler")
	err := r.ParseMultipartForm(104857600) // Up to 100MB stored in memory.
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	formdata := r.MultipartForm

	if val, ok := formdata.Value["Status"]; ok {
		writeUploadResponse(&w, types.CodeType_OK, val[0]);
		return
	}

	txString, ok := formdata.Value["transaction"]
	if !ok {
		writeUploadResponse(&w, types.CodeType_BCFSInvalidInput, "Missing transaction parameter.");
		return
	}

	stx := &crypto.SignedStruct{Base: &types.Transaction{Data:&types.RequestUpload{}}}
	var tx *types.Transaction
	if err := json.Unmarshal([]byte(txString[0]), stx); err != nil {
		writeUploadResponse(&w, types.CodeType_BCFSInvalidInput, "Could not Marshal transaction (SignedTransaction)");
		return
	} else if tx, ok = stx.Base.(*types.Transaction); !ok {
		writeUploadResponse(&w, types.CodeType_BCFSInvalidInput, "Could not Marshal transaction (Transaction)");
		return
	}
	reqUpload, ok := tx.Data.(*types.RequestUpload);
	if  !ok {
		writeUploadResponse(&w, types.CodeType_BCFSInvalidInput, "Could not Marshal transaction (RequestUpload)");
		return
	}

	fmt.Printf("Received tranc: %+v\n", stx)
	fmt.Printf("Received tranc: %+v\n", tx)
	fmt.Printf("Received tranc: %+v\n", tx.Data)
	fmt.Printf("Received tranc: %+v\n", reqUpload)
	fmt.Printf("Hash of tranc: %v\n", crypto.HashStruct(tx))


	// Check for replay attack
	txHash := crypto.HashStruct(tx)
	if app.HasSeenTranc(txHash) {
		writeUploadResponse(&w, types.CodeType_BadNonce, "Could not process transaction. Possible replay attack.");
		return
	}

	// Get signers identity and public key
	signer, pubKey := app.GetIdentityPublicKey(tx.Identity)
	if signer == nil {
		writeUploadResponse(&w, types.CodeType_Unauthorized, "Could not get access list")
		return
	}

	// Check if public key exists and if message is signed.
	if pubKey == nil {
		writeUploadResponse(&w, types.CodeType_BCFSInvalidSignature, "Could not locate public key");
		return
	} else if !stx.Verify(pubKey) {
		writeUploadResponse(&w, types.CodeType_BCFSInvalidSignature, "Could not verify signature");
		return
	}

	// Only type Client is allowed to upload data
	if signer.Type != conf.Client {
		writeUploadResponse(&w, types.CodeType_Unauthorized, "Only registered clients can upload data.");
		return
	}

	// Check if data hash is contained within the transaction.
	//reqUpload, ok := tx.Data.(types.RequestUpload)
	if reqUpload.Cid == "" || !ok {
		writeUploadResponse(&w, types.CodeType_BCFSInvalidInput, "Missing data hash parameter.");
		return
	}

	// Check that exactly one file is sent
	files, ok := formdata.File["file"]
	if !ok || len(files) > 1 {
		writeUploadResponse(&w, types.CodeType_BCFSInvalidInput, "File parameter should contain exactly one file.");
		return
	}

	// Try to open the file sent
	file := files[0]
	fopen, err := file.Open()
	defer fopen.Close()
	if err != nil {
		writeUploadResponse(&w, types.CodeType_InternalError, "Could not access attached file.");
		return
	}

	// Check if the hash of the upload file equals the hash contained in the transaction
	fileBytes, err := ioutil.ReadAll(fopen);
	if err != nil {
		writeUploadResponse(&w, types.CodeType_BCFSInvalidInput, "Could not get byte array of input file.");
		return
	} else if uplFileHash, err := crypto.IPFSHashData(fileBytes); err != nil {
		writeUploadResponse(&w, types.CodeType_BCFSInvalidInput, "Could not get hash of input file.");
		return
	} else if uplFileHash != reqUpload.Cid {
		writeUploadResponse(&w, types.CodeType_BCFSInvalidInput, "Hash of input file not present in transaction.");
		return
	}

	// Create file on disk in temporary storage.
	// TODO: Needed to detect whether the IPFS or Client is acting byzantine. This functionality is however not implemented.
	out, err := os.Create(conf.AppConfig().TempUploadPath + reqUpload.Cid)
	defer out.Close()
	if err != nil {
		writeUploadResponse(&w, types.CodeType_Unauthorized, "Unable to create the file for writing. Check your write access privilege");
		return
	} else if _, err = io.Copy(out, fopen); err != nil {
		writeUploadResponse(&w, types.CodeType_InternalError, err.Error());
		return
	}

	// Generate Sample of data. Distribute it to other TM nodes
	sample := crypto.GenerateStorageSample(&fileBytes)
	signedSample, err := sample.SignSample(app.privKey)
	if err != nil {
		writeUploadResponse(&w, types.CodeType_BCFSInvalidSignature, "Could not sign storage sample.");
		return
	}

	// Broadcast the signed storage sample to all other tendermint consensus nodes.
	// TODO: Add some mechanic to resend the sample to the nodes which are unavailble. Use the multicastQuery function.
	bytearr, err := json.Marshal(signedSample)
	responseChan := make(chan *QueryBroadcastReponse, len(app.TMRpcClients))
	done := make(chan struct{}, 1)
	app.broadcastQuery("/newsample", &bytearr, responseChan, done)
	goodResponses := make(map[string]bool)


	numResponses := len(app.TMRpcClients)
	var wg sync.WaitGroup
	wg.Add(numResponses)

	go func(){
		defer close(responseChan)
		defer close(done)
		for {
			select {
			case v := <-responseChan:
				fmt.Printf("Responsechan: %+v\n", v)
				if v.Err == nil && v.Result.Response.Code == uint32(types.CodeType_OK){
					goodResponses[v.Identity] = true
				}
				wg.Done()
				numResponses--
			case <-time.After(time.Duration(conf.AppConfig().TmQueryTimeoutSeconds) * time.Second):
				fmt.Println("TIMEOUT")
				wg.Add(-numResponses)
				return
			}
		}
	}()
	fmt.Println("Waiting for routines to finish.")
	wg.Wait()
	fmt.Println("Done waiting.")
	/*
	if _, ok := goodResponses[app.fingerprint]; !ok {
		// Could not store the sample in my own node? Some serious trouble.
		writeUploadResponse(&w, types.CodeType_OK, "Problems storing Storage sample.");
		return
	}*/

	// Need 2/3+ precommits to make progress. Quorum >= 2f + 1 = (2*n+1)/3
	if len(goodResponses) >= ((2*len(conf.AppConfig().TendermintNodes))+1)/3 {
		writeUploadResponse(&w, types.CodeType_OK, "File temporary stored and storage sample distributed. " +
			"After uploading file to IPFS, send a transaction to the mempool.");
		// Add transaction to list of known transactions
		app.seenTranc[txHash] = true
	} else {
		writeUploadResponse(&w, types.CodeType_Unauthorized, "Could not distribute storage samples to enough nodes" +
			" within the time period. Try again later.");
	}
}
