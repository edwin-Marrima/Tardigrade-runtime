#!/bin/bash
INITRAMFS_WORK_DIR="${INITRAMFS_WORK_DIR:-/tmp/initramfs}"
OUT_FILE="${OUT_FILE:-initrd.cpio}"


# Create directories
for dir in bin dev etc home mnt proc sys usr; do
    mkdir -p ${INITRAMFS_WORK_DIR}/$dir
done

cp busybox ${INITRAMFS_WORK_DIR}/bin/busybox
cp initramfs.sh ${INITRAMFS_WORK_DIR}/init
ls $INITRAMFS_WORK_DIR
chmod +x ${INITRAMFS_WORK_DIR}/init

# Create initramfs image
pushd ${INITRAMFS_WORK_DIR} > /dev/null
find . -print0 | cpio --null --create --verbose --format=newc > /tmp/initrd.cpio
popd > /dev/null

mv /tmp/initrd.cpio ${OUT_FILE}
ls -lha /tmp/initrd.cpio
chmod +x ${OUT_FILE}
rm -rf /tmp/initrd.cpio
rm -rf ${INITRAMFS_WORK_DIR}

ls -lah ${OUT_FILE}
echo ${OUT_FILE}