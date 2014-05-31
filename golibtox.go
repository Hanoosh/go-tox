package golibtox

// WIP - organ 2014

/*
#cgo LDFLAGS: -ltoxcore

#include <tox/tox.h>
#include <stdlib.h>

// Convenient macro:
// Creates the C function to directly register a given callback
#define HOOK(x) \
static void set_##x(Tox *tox, void *t) { \
	tox_##x(tox, hook_##x, t); \
}

void hook_callback_friend_request(Tox*, uint8_t*, uint8_t*, uint16_t, void*);
void hook_callback_friend_message(Tox*, int32_t, uint8_t*, uint16_t, void*);
void hook_callback_friend_action(Tox*, int32_t, uint8_t*, uint16_t, void*);
void hook_callback_name_change(Tox*, int32_t, uint8_t*, uint16_t, void*);
void hook_callback_status_message(Tox*, int32_t, uint8_t*, uint16_t, void*);
void hook_callback_user_status(Tox*, int32_t, uint8_t, void*);
void hook_callback_typing_change(Tox*, int32_t, uint8_t, void*);
void hook_callback_read_receipt(Tox*, int32_t, uint32_t, void*);
void hook_callback_connection_status(Tox*, int32_t, uint8_t, void*);
void hook_callback_file_send_request(Tox*, int32_t, uint8_t, uint64_t, uint8_t*, uint16_t, void*);
void hook_callback_file_control(Tox*, int32_t, uint8_t, uint8_t, uint8_t, uint8_t*, uint16_t, void*);
void hook_callback_file_data(Tox*, int32_t, uint8_t, uint8_t*, uint16_t, void*);

HOOK(callback_friend_request)
HOOK(callback_friend_message)
HOOK(callback_friend_action)
HOOK(callback_name_change)
HOOK(callback_status_message)
HOOK(callback_user_status)
HOOK(callback_typing_change)
HOOK(callback_read_receipt)
HOOK(callback_connection_status)
HOOK(callback_file_send_request)
HOOK(callback_file_control)
HOOK(callback_file_data)

*/
import "C"

import (
	"encoding/hex"
	"sync"
	"time"
	"unsafe"
)

type OnFriendRequest func(tox *Tox, publicKey []byte, data []byte, length uint16)
type OnFriendMessage func(tox *Tox, friendnumber int32, message []byte, length uint16)
type OnFriendAction func(tox *Tox, friendnumber int32, action []byte, length uint16)
type OnNameChange func(tox *Tox, friendnumber int32, name []byte, length uint16)
type OnStatusMessage func(tox *Tox, friendnumber int32, status []byte, length uint16)
type OnUserStatus func(tox *Tox, friendnumber int32, userstatus UserStatus)
type OnTypingChange func(tox *Tox, friendnumber int32, typing bool)
type OnReadReceipt func(tox *Tox, friendnumber int32, receipt uint32)
type OnConnectionStatus func(tox *Tox, friendnumber int32, online bool)
type OnFileSendRequest func(tox *Tox, friendnumber int32, filenumber uint8, filesize uint64, filename []byte, filenameLength uint16)
type OnFileControl func(tox *Tox, friendnumber int32, sending bool, filenumber uint8, fileControl FileControl, data []byte, length uint16)
type OnFileData func(tox *Tox, friendnumber int32, filenumber uint8, data []byte, length uint16)

type Tox struct {
	tox *C.struct_Tox
	mtx sync.Mutex
	// Callbacks
	onFriendRequest    OnFriendRequest
	onFriendMessage    OnFriendMessage
	onFriendAction     OnFriendAction
	onNameChange       OnNameChange
	onStatusMessage    OnStatusMessage
	onUserStatus       OnUserStatus
	onTypingChange     OnTypingChange
	onReadReceipt      OnReadReceipt
	onConnectionStatus OnConnectionStatus
	onFileSendRequest  OnFileSendRequest
	onFileControl      OnFileControl
	onFileData         OnFileData
}

func New() (*Tox, error) {
	ctox := C.tox_new(ENABLE_IPV6_DEFAULT)
	if ctox == nil {
		return nil, ErrInit
	}

	t := &Tox{tox: ctox}

	return t, nil
}

func (t *Tox) Kill() error {
	if t.tox == nil {
		return ErrBadTox
	}
	C.tox_kill(t.tox)

	return nil
}

func (t *Tox) Do() error {
	if t.tox == nil {
		return ErrBadTox
	}

	t.mtx.Lock()
	C.tox_do(t.tox)
	t.mtx.Unlock()

	return nil
}

func (t *Tox) BootstrapFromAddress(address string, port uint16, hexPublicKey string) error {
	if t.tox == nil {
		return ErrBadTox
	}

	caddr := C.CString(address)
	defer C.free(unsafe.Pointer(caddr))

	pubkey, err := hex.DecodeString(hexPublicKey)

	if err != nil {
		return err
	}

	C.tox_bootstrap_from_address(t.tox, caddr, ENABLE_IPV6_DEFAULT, C.htons((C.uint16_t)(port)), (*C.uint8_t)(&pubkey[0]))

	return nil
}

func (t *Tox) IsConnected() (bool, error) {
	if t.tox == nil {
		return false, ErrBadTox
	}

	return (C.tox_isconnected(t.tox) == 1), nil
}

func (t *Tox) GetAddress() ([]byte, error) {
	if t.tox == nil {
		return nil, ErrBadTox
	}

	address := make([]byte, FRIEND_ADDRESS_SIZE)
	C.tox_get_address(t.tox, (*C.uint8_t)(&address[0]))

	return address, nil
}

func (t *Tox) AddFriend(address []byte, data []byte) (int32, error) {
	if t.tox == nil {
		return 0, ErrBadTox
	}

	if len(address) != FRIEND_ADDRESS_SIZE {
		return 0, ErrArgs
	}

	ret := C.tox_add_friend(t.tox, (*C.uint8_t)(&address[0]), (*C.uint8_t)(&data[0]), (C.uint16_t)(len(data)))

	var faerr error

	switch FriendAddError(ret) {
	case FAERR_TOOLONG:
		faerr = FaerrTooLong
	case FAERR_NOMESSAGE:
		faerr = FaerrNoMessage
	case FAERR_OWNKEY:
		faerr = FaerrOwnKey
	case FAERR_ALREADYSENT:
		faerr = FaerrAlreadySent
	case FAERR_UNKNOWN:
		faerr = FaerrUnkown
	case FAERR_BADCHECKSUM:
		faerr = FaerrBadChecksum
	case FAERR_SETNEWNOSPAM:
		faerr = FaerrSetNewNospam
	case FAERR_NOMEM:
		faerr = FaerrNoMem
	}

	return int32(ret), faerr
}

func (t *Tox) AddFriendNorequest(clientId []byte) (int32, error) {
	if t.tox == nil {
		return -1, ErrBadTox
	}

	if len(clientId) != CLIENT_ID_SIZE {
		return -1, ErrArgs
	}

	n := C.tox_add_friend_norequest(t.tox, (*C.uint8_t)(&clientId[0]))
	if n == -1 {
		return -1, ErrFuncFail
	}

	return int32(n), nil
}

func (t *Tox) GetFriendNumber(clientId []byte) (int32, error) {
	if t.tox == nil {
		return -1, ErrBadTox
	}
	n := C.tox_get_friend_number(t.tox, (*C.uint8_t)(&clientId[0]))

	return int32(n), nil
}

func (t *Tox) GetClientId(friendnumber int32) ([]byte, error) {
	if t.tox == nil {
		return nil, ErrBadTox
	}
	clientId := make([]byte, CLIENT_ID_SIZE)
	ret := C.tox_get_client_id(t.tox, (C.int32_t)(friendnumber), (*C.uint8_t)(&clientId[0]))

	if ret != 0 {
		return nil, ErrFuncFail
	}

	return clientId, nil
}

func (t *Tox) DelFriend(friendnumber int32) error {
	if t.tox == nil {
		return ErrBadTox
	}
	ret := C.tox_del_friend(t.tox, (C.int32_t)(friendnumber))

	if ret != 0 {
		return ErrFuncFail
	}

	return nil
}

func (t *Tox) GetFriendConnectionStatus(friendnumber int32) (bool, error) {
	if t.tox == nil {
		return false, ErrBadTox
	}
	ret := C.tox_get_friend_connection_status(t.tox, (C.int32_t)(friendnumber))
	if ret == -1 {
		return false, ErrFuncFail
	}

	return (int(ret) == 1), nil
}

func (t *Tox) FriendExists(friendnumber int32) (bool, error) {
	if t.tox == nil {
		return false, ErrBadTox
	}
	//int tox_friend_exists(Tox *tox, int32_t friendnumber);
	ret := C.tox_friend_exists(t.tox, (C.int32_t)(friendnumber))

	return (int(ret) == 1), nil
}

func (t *Tox) SendMessage(friendnumber int32, message []byte) (uint32, error) {
	if t.tox == nil {
		return 0, ErrBadTox
	}

	n := C.tox_send_message(t.tox, (C.int32_t)(friendnumber), (*C.uint8_t)(&message[0]), (C.uint32_t)(len(message)))
	if n == 0 {
		return 0, ErrFuncFail
	}

	return uint32(n), nil
}

func (t *Tox) SendMessageWithId(friendnumber int32, id uint32, message []byte) (uint32, error) {
	if t.tox == nil {
		return 0, ErrBadTox
	}

	n := C.tox_send_message_withid(t.tox, (C.int32_t)(friendnumber), (C.uint32_t)(id), (*C.uint8_t)(&message[0]), (C.uint32_t)(len(message)))
	if n == 0 {
		return 0, ErrFuncFail
	}

	return uint32(n), nil
}

func (t *Tox) SendAction(friendnumber int32, action []byte) (uint32, error) {
	if t.tox == nil {
		return 0, ErrBadTox
	}

	n := C.tox_send_action(t.tox, (C.int32_t)(friendnumber), (*C.uint8_t)(&action[0]), (C.uint32_t)(len(action)))
	if n == 0 {
		return 0, ErrFuncFail
	}

	return uint32(n), nil
}

func (t *Tox) SendActionWithId(friendnumber int32, id uint32, action []byte) (uint32, error) {
	if t.tox == nil {
		return 0, ErrBadTox
	}

	n := C.tox_send_message_withid(t.tox, (C.int32_t)(friendnumber), (C.uint32_t)(id), (*C.uint8_t)(&action[0]), (C.uint32_t)(len(action)))
	if n == 0 {
		return 0, ErrFuncFail
	}

	return uint32(n), nil
}

func (t *Tox) SetName(name string) error {
	if t.tox == nil {
		return ErrBadTox
	}

	ret := C.tox_set_name(t.tox, (*C.uint8_t)(&[]byte(name)[0]), (C.uint16_t)(len(name)))
	if ret != 0 {
		return ErrFuncFail
	}

	return nil
}

func (t *Tox) GetSelfName() (string, error) {
	if t.tox == nil {
		return "", ErrBadTox
	}

	cname := make([]byte, MAX_NAME_LENGTH)

	n := C.tox_get_self_name(t.tox, (*C.uint8_t)(&cname[0]))
	if n == 0 {
		return "", ErrFuncFail
	}

	name := string(cname[:n])

	return name, nil
}

func (t *Tox) GetName(friendnumber int32) (string, error) {
	if t.tox == nil {
		return "", ErrBadTox
	}

	cname := make([]byte, MAX_NAME_LENGTH)

	n := C.tox_get_name(t.tox, (C.int32_t)(friendnumber), (*C.uint8_t)(&cname[0]))
	if n == -1 {
		return "", ErrFuncFail
	}

	name := string(cname[:n])

	return name, nil
}

func (t *Tox) GetNameSize(friendnumber int32) (int, error) {
	if t.tox == nil {
		return -1, ErrBadTox
	}

	ret := C.tox_get_name_size(t.tox, (C.int32_t)(friendnumber))
	if ret == -1 {
		return -1, ErrFuncFail
	}

	return int(ret), nil
}

func (t *Tox) GetSelfNameSize() (int, error) {
	if t.tox == nil {
		return -1, ErrBadTox
	}

	ret := C.tox_get_self_name_size(t.tox)
	if ret == -1 {
		return -1, ErrFuncFail
	}

	return int(ret), nil
}

func (t *Tox) SetStatusMessage(status []byte) error {
	if t.tox == nil {
		return ErrBadTox
	}

	ret := C.tox_set_status_message(t.tox, (*C.uint8_t)(&status[0]), (C.uint16_t)(len(status)))
	if ret != 0 {
		return ErrFuncFail
	}

	return nil
}

func (t *Tox) SetUserStatus(userstatus UserStatus) error {
	if t.tox == nil {
		return ErrBadTox
	}

	ret := C.tox_set_user_status(t.tox, (C.uint8_t)(userstatus))
	if ret != 0 {
		return ErrFuncFail
	}

	return nil
}

func (t *Tox) GetStatusMessageSize(friendnumber int32) (int, error) {
	if t.tox == nil {
		return -1, ErrBadTox
	}

	ret := C.tox_get_status_message_size(t.tox, (C.int32_t)(friendnumber))
	if ret == -1 {
		return -1, ErrFuncFail
	}

	return int(ret), nil
}

func (t *Tox) GetSelfStatusMessageSize() (int, error) {
	if t.tox == nil {
		return -1, ErrBadTox
	}

	ret := C.tox_get_self_status_message_size(t.tox)
	if ret == -1 {
		return -1, ErrFuncFail
	}

	return int(ret), nil
}

func (t *Tox) GetStatusMessage(friendnumber int32) ([]byte, error) {
	if t.tox == nil {
		return nil, ErrBadTox
	}

	status := make([]byte, MAX_STATUSMESSAGE_LENGTH)

	n := C.tox_get_status_message(t.tox, (C.int32_t)(friendnumber), (*C.uint8_t)(&status[0]), MAX_STATUSMESSAGE_LENGTH)
	if n == -1 {
		return nil, ErrFuncFail
	}

	// Truncate status to n-byte read
	status = status[:n]

	return status, nil
}

func (t *Tox) GetSelfStatusMessage() ([]byte, error) {
	if t.tox == nil {
		return nil, ErrBadTox
	}

	status := make([]byte, MAX_STATUSMESSAGE_LENGTH)

	n := C.tox_get_self_status_message(t.tox, (*C.uint8_t)(&status[0]), MAX_STATUSMESSAGE_LENGTH)
	if n == -1 {
		return nil, ErrFuncFail
	}

	// Truncate status to n-byte read
	status = status[:n]

	return status, nil
}

func (t *Tox) GetUserStatus(friendnumber int32) (UserStatus, error) {
	if t.tox == nil {
		return USERSTATUS_INVALID, ErrBadTox
	}
	n := C.tox_get_user_status(t.tox, (C.int32_t)(friendnumber))

	return UserStatus(n), nil
}

func (t *Tox) GetSelfUserStatus() (UserStatus, error) {
	if t.tox == nil {
		return USERSTATUS_INVALID, ErrBadTox
	}
	n := C.tox_get_self_user_status(t.tox)

	return UserStatus(n), nil
}

func (t *Tox) GetLastOnline(friendnumber int32) (time.Time, error) {
	if t.tox == nil {
		return time.Time{}, ErrBadTox
	}

	ret := C.tox_get_last_online(t.tox, (C.int32_t)(friendnumber))

	if int(ret) == -1 {
		return time.Time{}, ErrFuncFail
	}

	last := time.Unix(int64(ret), 0)

	return last, nil
}

func (t *Tox) SetUserIsTyping(friendnumber int32, typing bool) error {
	if t.tox == nil {
		return ErrBadTox
	}
	ctyping := 0
	if typing {
		ctyping = 1
	}

	ret := C.tox_set_user_is_typing(t.tox, (C.int32_t)(friendnumber), (C.uint8_t)(ctyping))

	if ret != 0 {
		return ErrFuncFail
	}

	return nil
}

func (t *Tox) GetIsTyping(friendnumber int32) (bool, error) {
	if t.tox == nil {
		return false, ErrBadTox
	}

	ret := C.tox_get_is_typing(t.tox, (C.int32_t)(friendnumber))

	return (ret == 1), nil
}

func (t *Tox) SetSendsReceipts(friendnumber int32, send bool) error {
	if t.tox == nil {
		return ErrBadTox
	}
	csend := 0
	if send {
		csend = 1
	}

	C.tox_set_sends_receipts(t.tox, (C.int32_t)(friendnumber), (C.int)(csend))

	return nil
}

func (t *Tox) CountFriendlist() (uint32, error) {
	if t.tox == nil {
		return 0, ErrBadTox
	}
	n := C.tox_count_friendlist(t.tox)

	return uint32(n), nil
}

func (t *Tox) GetNumOnlineFriends() (uint32, error) {
	if t.tox == nil {
		return 0, ErrBadTox
	}
	n := C.tox_get_num_online_friends(t.tox)

	return uint32(n), nil
}

func (t *Tox) GetFriendlist() ([]int32, error) {
	if t.tox == nil {
		return nil, ErrBadTox
	}

	size, _ := t.CountFriendlist()
	cfriendlist := make([]int32, size)

	n := C.tox_get_friendlist(t.tox, (*C.int32_t)(&cfriendlist[0]), (C.uint32_t)(size))

	friendlist := cfriendlist[:n]

	return friendlist, nil
}

func (t *Tox) GetNospam() (uint32, error) {
	if t.tox == nil {
		return 0, ErrBadTox
	}

	n := C.tox_get_nospam(t.tox)

	return uint32(n), nil
}

func (t *Tox) SetNospam(nospam uint32) error {
	if t.tox == nil {
		return ErrBadTox
	}

	C.tox_set_nospam(t.tox, (C.uint32_t)(nospam))

	return nil
}

func (t *Tox) NewFileSender(friendnumber int32, filesize uint64, filename []byte) (int, error) {
	if t.tox == nil {
		return -1, ErrBadTox
	}

	if len(filename) > 255 {
		return -1, ErrArgs
	}

	n := C.tox_new_file_sender(t.tox, (C.int32_t)(friendnumber), (C.uint64_t)(filesize), (*C.uint8_t)(&filename[0]), (C.uint16_t)(len(filename)))

	if n == -1 {
		return -1, ErrFuncFail
	}

	return int(n), nil
}

func (t *Tox) FileSendControl(friendnumber int32, receiving bool, filenumber uint8, messageId FileControl, data []byte) error {
	if t.tox == nil {
		return ErrBadTox
	}

	cReceiving := 0
	if receiving {
		cReceiving = 1
	}

	// Stupid workaround to prevent index out of range when using &data[0] if data == nil
	var cdata *C.uint8_t
	var clen C.uint16_t

	if data == nil {
		cdata = nil
		clen = 0
	} else {
		cdata = (*C.uint8_t)(&data[0])
		clen = (C.uint16_t)(len(data))
	}
	// End of stupid workaround

	n := C.tox_file_send_control(t.tox, (C.int32_t)(friendnumber), (C.uint8_t)(cReceiving), (C.uint8_t)(filenumber), (C.uint8_t)(messageId), cdata, clen)

	if n == -1 {
		return ErrFuncFail
	}

	return nil
}

func (t *Tox) FileSendData(friendnumber int32, filenumber uint8, data []byte) error {
	if t.tox == nil {
		return ErrBadTox
	}

	if len(data) == 0 {
		return ErrArgs

	}

	n := C.tox_file_send_data(t.tox, (C.int32_t)(friendnumber), (C.uint8_t)(filenumber), (*C.uint8_t)(&data[0]), (C.uint16_t)(len(data)))

	if n == -1 {
		return ErrFuncFail
	}

	return nil
}

func (t *Tox) FileDataSize(friendnumber int32) (int, error) {
	if t.tox == nil {
		return -1, ErrBadTox
	}

	n := C.tox_file_data_size(t.tox, (C.int32_t)(friendnumber))

	if n == -1 {
		return -1, ErrFuncFail
	}

	return int(n), nil
}

func (t *Tox) FileDataRemaining(friendnumber int32, filenumber uint8, receiving bool) (uint64, error) {
	if t.tox == nil {
		return 0, ErrBadTox
	}

	cReceiving := 0
	if receiving {
		cReceiving = 1
	}

	n := C.tox_file_data_remaining(t.tox, (C.int32_t)(friendnumber), (C.uint8_t)(filenumber), (C.uint8_t)(cReceiving))

	if n == 0 {
		return 0, ErrFuncFail
	}

	return uint64(n), nil
}

func (t *Tox) Size() (uint32, error) {
	if t.tox == nil {
		return 0, ErrBadTox
	}

	return uint32(C.tox_size(t.tox)), nil
}

func (t *Tox) Save() ([]byte, error) {
	if t.tox == nil {
		return nil, ErrBadTox
	}
	size, _ := t.Size()

	data := make([]byte, size)
	C.tox_save(t.tox, (*C.uint8_t)(&data[0]))

	return data, nil
}

func (t *Tox) Load(data []byte) error {
	if t.tox == nil {
		return ErrBadTox
	}

	ret := C.tox_load(t.tox, (*C.uint8_t)(&data[0]), (C.uint32_t)(len(data)))

	if ret == -1 {
		return ErrFuncFail
	}

	return nil
}

func (t *Tox) CallbackFriendRequest(f OnFriendRequest) {
	if t.tox != nil {
		t.onFriendRequest = f
		C.set_callback_friend_request(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackFriendMessage(f OnFriendMessage) {
	if t.tox != nil {
		t.onFriendMessage = f
		C.set_callback_friend_message(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackFriendAction(f OnFriendAction) {
	if t.tox != nil {
		t.onFriendAction = f
		C.set_callback_friend_action(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackNameChange(f OnNameChange) {
	if t.tox != nil {
		t.onNameChange = f
		C.set_callback_name_change(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackStatusMessage(f OnStatusMessage) {
	if t.tox != nil {
		t.onStatusMessage = f
		C.set_callback_status_message(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackUserStatus(f OnUserStatus) {
	if t.tox != nil {
		t.onUserStatus = f
		C.set_callback_user_status(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackTypingChange(f OnTypingChange) {
	if t.tox != nil {
		t.onTypingChange = f
		C.set_callback_typing_change(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackReadReceipt(f OnReadReceipt) {
	if t.tox != nil {
		t.onReadReceipt = f
		C.set_callback_read_receipt(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackConnectionStatus(f OnConnectionStatus) {
	if t.tox != nil {
		t.onConnectionStatus = f
		C.set_callback_connection_status(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackFileSendRequest(f OnFileSendRequest) {
	if t.tox != nil {
		t.onFileSendRequest = f
		C.set_callback_file_send_request(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackFileControl(f OnFileControl) {
	if t.tox != nil {
		t.onFileControl = f
		C.set_callback_file_control(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackFileData(f OnFileData) {
	if t.tox != nil {
		t.onFileData = f
		C.set_callback_file_data(t.tox, unsafe.Pointer(t))
	}
}
